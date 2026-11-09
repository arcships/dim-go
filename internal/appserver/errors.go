package appserver

import (
	"encoding/json"
	"errors"
)

// Daemon appError values from docs/protocol-contract.md §13. Code -32602 has
// no appError name; it is classified as BAD_REQUEST by code alone.
const (
	AppErrorUnknownMethod           = "unknown_method"
	AppErrorServiceReloading        = "service_reloading"
	AppErrorConnectionBusy          = "connection_busy"
	AppErrorServiceUnavailable      = "service_unavailable"
	AppErrorSubscriptionChanged     = "subscription_changed"
	AppErrorSessionClosed           = "session_closed"
	AppErrorSessionActive           = "session_active"
	AppErrorProtocolVersion         = "protocol_version"
	AppErrorSessionProtocolVersion  = "session_protocol_version"
	AppErrorSubmissionUnknown       = "submission_unknown"
	AppErrorSubmissionAmbiguous     = "submission_ambiguous"
	AppErrorPermissionConflict      = "permission_conflict"
	AppErrorPermissionRulesConflict = "permission_rules_conflict"
	AppErrorQuestionNotPending      = "question_not_pending"
	AppErrorOperationInterrupted    = "operation_interrupted"
	AppErrorLocalTransportRequired  = "local_transport_required"
	AppErrorHistoryTooLarge         = "history_too_large"
	AppErrorBranchEnded             = "branch_ended"
	AppErrorAttachmentInvalid       = "attachment_invalid"
	AppErrorAttachmentTooLarge      = "attachment_too_large"
	AppErrorInternal                = "internal"
)

// ErrorClass groups §13 rows by the action they require.
type ErrorClass int

const (
	ErrorClassInternal ErrorClass = iota
	ErrorClassBadRequest
	ErrorClassCapability
	ErrorClassTransient
	ErrorClassUnavailable
	ErrorClassReplayGap
	ErrorClassSessionClosed
	ErrorClassSessionActive
	ErrorClassProtocolIncompatible
	ErrorClassSubmissionUnknown
	ErrorClassSubmissionAmbiguous
	ErrorClassPermissionConflict
	ErrorClassQuestionSettled
	ErrorClassInterrupted
	ErrorClassLocalTransport
	ErrorClassHistoryTooLarge
	ErrorClassBranchEnded
)

func (c ErrorClass) String() string {
	switch c {
	case ErrorClassBadRequest:
		return "bad_request"
	case ErrorClassCapability:
		return "capability"
	case ErrorClassTransient:
		return "transient"
	case ErrorClassUnavailable:
		return "unavailable"
	case ErrorClassReplayGap:
		return "replay_gap"
	case ErrorClassSessionClosed:
		return "session_closed"
	case ErrorClassSessionActive:
		return "session_active"
	case ErrorClassProtocolIncompatible:
		return "protocol_incompatible"
	case ErrorClassSubmissionUnknown:
		return "submission_unknown"
	case ErrorClassSubmissionAmbiguous:
		return "submission_ambiguous"
	case ErrorClassPermissionConflict:
		return "permission_conflict"
	case ErrorClassQuestionSettled:
		return "question_settled"
	case ErrorClassInterrupted:
		return "interrupted"
	case ErrorClassLocalTransport:
		return "local_transport"
	case ErrorClassHistoryTooLarge:
		return "history_too_large"
	case ErrorClassBranchEnded:
		return "branch_ended"
	default:
		return "internal"
	}
}

// MappedError is the §13 interpretation of a daemon error. AppError is empty
// when the class comes from the JSON-RPC code alone (-32602 BAD_REQUEST),
// because the contract defines no appError name for that case; use Class.
//
// ReadOnlyRetryOnce is the subscription_changed replay-gap rule: a read-only
// request may re-attach and retry exactly once, while mutations are never
// replayed automatically.
type MappedError struct {
	AppError          string
	Class             ErrorClass
	Retryable         bool
	ReadOnlyRetryOnce bool
}

func (m MappedError) Error() string {
	if m.AppError == "" {
		return "appserver: bad_request (" + m.Class.String() + ")"
	}
	return "appserver: " + m.AppError + " (" + m.Class.String() + ")"
}

// MapRPCError applies §13's precedence: code -32602 always means invalid
// params; otherwise error.data.appError selects the class, and an unknown or
// missing appError degrades to a non-retryable INTERNAL.
func MapRPCError(err *RPCError) MappedError {
	if err == nil {
		return internalError()
	}
	if err.Code == -32602 {
		// The contract has no appError name for invalid params; the class is
		// derived from the JSON-RPC code.
		return MappedError{Class: ErrorClassBadRequest}
	}
	switch rpcAppError(err.Data) {
	case AppErrorUnknownMethod:
		return MappedError{AppError: AppErrorUnknownMethod, Class: ErrorClassCapability}
	case AppErrorServiceReloading:
		return MappedError{AppError: AppErrorServiceReloading, Class: ErrorClassTransient, Retryable: true}
	case AppErrorConnectionBusy:
		return MappedError{AppError: AppErrorConnectionBusy, Class: ErrorClassTransient, Retryable: true}
	case AppErrorServiceUnavailable:
		return MappedError{AppError: AppErrorServiceUnavailable, Class: ErrorClassUnavailable}
	case AppErrorSubscriptionChanged:
		return MappedError{AppError: AppErrorSubscriptionChanged, Class: ErrorClassReplayGap, ReadOnlyRetryOnce: true}
	case AppErrorSessionClosed:
		return MappedError{AppError: AppErrorSessionClosed, Class: ErrorClassSessionClosed}
	case AppErrorSessionActive:
		return MappedError{AppError: AppErrorSessionActive, Class: ErrorClassSessionActive}
	case AppErrorProtocolVersion:
		return MappedError{AppError: AppErrorProtocolVersion, Class: ErrorClassProtocolIncompatible}
	case AppErrorSessionProtocolVersion:
		return MappedError{AppError: AppErrorSessionProtocolVersion, Class: ErrorClassProtocolIncompatible}
	case AppErrorSubmissionUnknown:
		return MappedError{AppError: AppErrorSubmissionUnknown, Class: ErrorClassSubmissionUnknown}
	case AppErrorSubmissionAmbiguous:
		return MappedError{AppError: AppErrorSubmissionAmbiguous, Class: ErrorClassSubmissionAmbiguous}
	case AppErrorPermissionConflict:
		return MappedError{AppError: AppErrorPermissionConflict, Class: ErrorClassPermissionConflict}
	case AppErrorPermissionRulesConflict:
		return MappedError{AppError: AppErrorPermissionRulesConflict, Class: ErrorClassPermissionConflict}
	case AppErrorQuestionNotPending:
		return MappedError{AppError: AppErrorQuestionNotPending, Class: ErrorClassQuestionSettled}
	case AppErrorOperationInterrupted:
		return MappedError{AppError: AppErrorOperationInterrupted, Class: ErrorClassInterrupted}
	case AppErrorLocalTransportRequired:
		return MappedError{AppError: AppErrorLocalTransportRequired, Class: ErrorClassLocalTransport}
	case AppErrorHistoryTooLarge:
		return MappedError{AppError: AppErrorHistoryTooLarge, Class: ErrorClassHistoryTooLarge}
	case AppErrorBranchEnded:
		return MappedError{AppError: AppErrorBranchEnded, Class: ErrorClassBranchEnded}
	case AppErrorAttachmentInvalid, AppErrorAttachmentTooLarge:
		// Definitive rejections of the attachment payload: the same request
		// would fail again, so they map to a non-retryable bad request.
		return MappedError{AppError: rpcAppError(err.Data), Class: ErrorClassBadRequest}
	default:
		return internalError()
	}
}

func internalError() MappedError {
	return MappedError{AppError: AppErrorInternal, Class: ErrorClassInternal}
}

// rpcAppError extracts error.data.appError without trusting the rest of data.
func rpcAppError(data json.RawMessage) string {
	if len(data) == 0 || string(data) == "null" {
		return ""
	}
	var payload struct {
		AppError string `json:"appError"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	return payload.AppError
}

// AsMappedError finds a daemon *RPCError in err and maps it. It returns false
// for client-side errors such as *LocalError.
func AsMappedError(err error, target *MappedError) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	*target = MapRPCError(rpcErr)
	return true
}

// IsAppError reports whether err carries appError, either as a client-side
// LocalError or as a daemon error after §13 mapping.
func IsAppError(err error, appError string) bool {
	var localErr *LocalError
	if errors.As(err, &localErr) {
		return localErr.AppError == appError
	}
	var mapped MappedError
	return AsMappedError(err, &mapped) && mapped.AppError == appError
}
