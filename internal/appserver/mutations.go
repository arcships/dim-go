package appserver

import (
	"context"
	"errors"
	"fmt"
)

// MutationMethod names the mutation methods that docs/protocol-contract.md
// defines outside the read helpers.
type MutationMethod string

const (
	MutationSessionSubmit       MutationMethod = "session.submit"
	MutationSessionStop         MutationMethod = "session.stop"
	MutationApprovalRespond     MutationMethod = "approval.respond"
	MutationSessionDelete       MutationMethod = "session.delete"
	MutationSessionBranchCreate MutationMethod = "session.branch_create"
	MutationSessionBranchEnd    MutationMethod = "session.branch_end"
	MutationAttachmentPut       MutationMethod = "attachment.put"
)

// ErrUnknownResult marks a mutation whose outcome is unknown: timeout,
// disconnect, or a daemon reply that says the submission cannot be resolved.
// Callers must read back or ask the user; they must never resend automatically
// (§14).
var ErrUnknownResult = errors.New("appserver: mutation result unknown")

// MutationError is a daemon reply that definitively rejected the mutation.
// AppError keeps the raw appError string, including values this client does not
// know, while Mapped holds the §13 classification. Use Unsupported to find the
// visible-disabled/WIP case.
type MutationError struct {
	Method   MutationMethod
	AppError string
	Mapped   MappedError
	Err      error
}

func (e *MutationError) Error() string {
	if e.Mapped.Class == ErrorClassCapability {
		return fmt.Sprintf("appserver: %s unavailable (%s): entry disabled (WIP)", e.Method, AppErrorUnknownMethod)
	}
	if e.AppError != "" {
		return fmt.Sprintf("appserver: %s: %s (%s)", e.Method, e.AppError, e.Mapped.Class)
	}
	return fmt.Sprintf("appserver: %s: %s", e.Method, e.Mapped.Class)
}

func (e *MutationError) Unwrap() error { return e.Err }

// Is lets errors.Is(err, ErrCapabilityUnavailable) find the missing-method case
// without matching a concrete appError string.
func (e *MutationError) Is(target error) bool {
	return target == ErrCapabilityUnavailable && e.Mapped.Class == ErrorClassCapability
}

// Unsupported reports whether the daemon answered unknown_method. The method is
// absent upstream: keep the entry point disabled and do not retry (§13).
func (e *MutationError) Unsupported() bool {
	return e.Mapped.Class == ErrorClassCapability
}

// UnknownResultError reports that a mutation may or may not have been applied.
// ReadBack is the contract-defined read used to resolve the doubt; Refresh means
// the caller should refresh the session snapshot instead of reading one input.
// Never resend the mutation (§14).
type UnknownResultError struct {
	Method   MutationMethod
	AppError string
	ReadBack string
	Refresh  bool
	Err      error
}

func (e *UnknownResultError) Error() string {
	message := fmt.Sprintf("appserver: %s: result unknown", e.Method)
	if e.AppError != "" {
		message += " (" + e.AppError + ")"
	}
	if e.ReadBack != "" {
		message += "; read back with " + e.ReadBack
	}
	if e.Refresh {
		message += "; refresh the session snapshot"
	}
	return message + "; do not resend"
}

func (e *UnknownResultError) Unwrap() error { return e.Err }

func (e *UnknownResultError) Is(target error) bool { return target == ErrUnknownResult }

// mutationReadBack names the read that resolves an unknown result. Every entry
// is an existing protocol method; no local state is consulted.
func mutationReadBack(method MutationMethod) string {
	switch method {
	case MutationSessionSubmit:
		return "session.input_lookup"
	case MutationSessionStop:
		return "session.state"
	case MutationApprovalRespond:
		// attach replays the pending approval set (§8).
		return "session.attach"
	case MutationSessionDelete:
		return "session.catalog"
	case MutationSessionBranchCreate, MutationSessionBranchEnd:
		// The directory shows whether a line appeared or reached ending/ended;
		// it cannot prove that a timed-out create did not happen.
		return "session.branches"
	case MutationAttachmentPut:
		// docs/protocol-contract.md §10: attachment has no list/get method, so
		// an unknown result can only be reported, never read back.
		return ""
	default:
		return ""
	}
}

// mapMutationError converts a Call error into the §13/§14 mutation view. A
// LocalError timeout, abort, or disconnect is an unknown result. Daemon
// submission_unknown/submission_ambiguous replies are also unknown results.
// Everything else is a definitive rejection and keeps the raw appError so an
// unknown value stays observable.
func mapMutationError(method MutationMethod, err error) error {
	var localErr *LocalError
	if errors.As(err, &localErr) {
		switch localErr.AppError {
		case "connection_closed", "request_timeout", "request_aborted":
			return &UnknownResultError{
				Method:   method,
				AppError: localErr.AppError,
				ReadBack: mutationReadBack(method),
				Err:      err,
			}
		default:
			return err
		}
	}

	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return err
	}
	appError := rpcAppError(rpcErr.Data)
	// Contract §13: "判定顺序：先看 code === -32602（一律 invalid params /
	// BAD_REQUEST），再按 appError 分派". -32602 is a definitive rejection even
	// when data.appError says submission_unknown, so it must never become an
	// unknown-result signal.
	if rpcErr.Code == -32602 {
		return &MutationError{
			Method:   method,
			AppError: appError,
			Mapped:   MapRPCError(rpcErr),
			Err:      err,
		}
	}
	if method == MutationSessionSubmit {
		switch appError {
		case AppErrorSubmissionUnknown:
			return &UnknownResultError{
				Method:   method,
				AppError: appError,
				ReadBack: mutationReadBack(method),
				Err:      err,
			}
		case AppErrorSubmissionAmbiguous:
			return &UnknownResultError{
				Method:   method,
				AppError: appError,
				Refresh:  true,
				Err:      err,
			}
		}
	}
	return &MutationError{
		Method:   method,
		AppError: appError,
		Mapped:   MapRPCError(rpcErr),
		Err:      err,
	}
}

// Submit sends session.submit and decodes the minimum {event} result.
//
// The contract defines no hello.features gate for the four core mutations, so
// this helper does not invent one. A daemon that lacks the method answers
// unknown_method, which is surfaced as a *MutationError matching
// ErrCapabilityUnavailable.
func Submit(ctx context.Context, client *Client, params SubmitParams) (SubmitResult, error) {
	if err := params.Validate(); err != nil {
		return SubmitResult{}, err
	}
	raw, err := client.Call(ctx, string(MutationSessionSubmit), params)
	if err != nil {
		return SubmitResult{}, mapMutationError(MutationSessionSubmit, err)
	}
	result, err := DecodeSubmitResult(raw)
	if err != nil {
		// The submission ran but its result does not satisfy the contract;
		// resolve it by reading the input back, never by resubmitting.
		return SubmitResult{}, &UnknownResultError{
			Method:   MutationSessionSubmit,
			ReadBack: mutationReadBack(MutationSessionSubmit),
			Err:      err,
		}
	}
	return result, nil
}

// Stop sends session.stop. The contract ignores the response body; the stopped
// state converges through session.state (§10).
func Stop(ctx context.Context, client *Client, params StopParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	if _, err := client.Call(ctx, string(MutationSessionStop), params); err != nil {
		return mapMutationError(MutationSessionStop, err)
	}
	return nil
}

// ApprovalRespondOutcome distinguishes a normal reply from the §12 case where
// the approval no longer exists.
type ApprovalRespondOutcome string

const (
	ApprovalOutcomeResponded ApprovalRespondOutcome = "responded"
	ApprovalOutcomeCancelled ApprovalRespondOutcome = "cancelled"
)

// ApprovalRespond sends approval.respond. A -32602 reply means the approval is
// already gone: the contract maps it to cancelled without an error or retry
// (§12).
func ApprovalRespond(ctx context.Context, client *Client, params ApprovalRespondParams) (ApprovalRespondOutcome, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}
	if _, err := client.Call(ctx, string(MutationApprovalRespond), params); err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32602 {
			return ApprovalOutcomeCancelled, nil
		}
		return "", mapMutationError(MutationApprovalRespond, err)
	}
	return ApprovalOutcomeResponded, nil
}

// Delete sends session.delete. The contract defines no response body, so a
// successful reply only means the request was accepted; callers that need to
// know whether the session disappeared must read session.catalog.
func Delete(ctx context.Context, client *Client, params DeleteParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	if _, err := client.Call(ctx, string(MutationSessionDelete), params); err != nil {
		return mapMutationError(MutationSessionDelete, err)
	}
	return nil
}
