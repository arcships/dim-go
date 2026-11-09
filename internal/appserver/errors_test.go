package appserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func appErrorData(appError string) json.RawMessage {
	return json.RawMessage(`{"appError":"` + appError + `"}`)
}

func mapped(appError string, class ErrorClass, retryable, readOnlyRetryOnce bool) MappedError {
	return MappedError{AppError: appError, Class: class, Retryable: retryable, ReadOnlyRetryOnce: readOnlyRetryOnce}
}

func TestMapRPCError(t *testing.T) {
	cases := []struct {
		name string
		err  *RPCError
		want MappedError
	}{
		{"nil", nil, mapped(AppErrorInternal, ErrorClassInternal, false, false)},
		{
			name: "-32602 takes precedence over appError",
			err:  &RPCError{Code: -32602, Message: "invalid params", Data: appErrorData(AppErrorUnknownMethod)},
			want: mapped("", ErrorClassBadRequest, false, false),
		},
		{
			name: "unknown_method",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorUnknownMethod)},
			want: mapped(AppErrorUnknownMethod, ErrorClassCapability, false, false),
		},
		{
			name: "service_reloading",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorServiceReloading)},
			want: mapped(AppErrorServiceReloading, ErrorClassTransient, true, false),
		},
		{
			name: "connection_busy",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorConnectionBusy)},
			want: mapped(AppErrorConnectionBusy, ErrorClassTransient, true, false),
		},
		{
			name: "service_unavailable",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorServiceUnavailable)},
			want: mapped(AppErrorServiceUnavailable, ErrorClassUnavailable, false, false),
		},
		{
			name: "subscription_changed",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSubscriptionChanged)},
			want: mapped(AppErrorSubscriptionChanged, ErrorClassReplayGap, false, true),
		},
		{
			name: "session_closed",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSessionClosed)},
			want: mapped(AppErrorSessionClosed, ErrorClassSessionClosed, false, false),
		},
		{
			name: "session_active",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSessionActive)},
			want: mapped(AppErrorSessionActive, ErrorClassSessionActive, false, false),
		},
		{
			name: "protocol_version",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorProtocolVersion)},
			want: mapped(AppErrorProtocolVersion, ErrorClassProtocolIncompatible, false, false),
		},
		{
			name: "session_protocol_version",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSessionProtocolVersion)},
			want: mapped(AppErrorSessionProtocolVersion, ErrorClassProtocolIncompatible, false, false),
		},
		{
			name: "submission_unknown",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSubmissionUnknown)},
			want: mapped(AppErrorSubmissionUnknown, ErrorClassSubmissionUnknown, false, false),
		},
		{
			name: "submission_ambiguous",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSubmissionAmbiguous)},
			want: mapped(AppErrorSubmissionAmbiguous, ErrorClassSubmissionAmbiguous, false, false),
		},
		{
			name: "permission_conflict",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorPermissionConflict)},
			want: mapped(AppErrorPermissionConflict, ErrorClassPermissionConflict, false, false),
		},
		{
			name: "permission_rules_conflict",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorPermissionRulesConflict)},
			want: mapped(AppErrorPermissionRulesConflict, ErrorClassPermissionConflict, false, false),
		},
		{
			name: "question_not_pending",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorQuestionNotPending)},
			want: mapped(AppErrorQuestionNotPending, ErrorClassQuestionSettled, false, false),
		},
		{
			name: "operation_interrupted",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorOperationInterrupted)},
			want: mapped(AppErrorOperationInterrupted, ErrorClassInterrupted, false, false),
		},
		{
			name: "local_transport_required",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorLocalTransportRequired)},
			want: mapped(AppErrorLocalTransportRequired, ErrorClassLocalTransport, false, false),
		},
		{
			name: "history_too_large",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorHistoryTooLarge)},
			want: mapped(AppErrorHistoryTooLarge, ErrorClassHistoryTooLarge, false, false),
		},
		{
			name: "branch_ended",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorBranchEnded)},
			want: mapped(AppErrorBranchEnded, ErrorClassBranchEnded, false, false),
		},
		{
			name: "attachment_invalid",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorAttachmentInvalid)},
			want: mapped(AppErrorAttachmentInvalid, ErrorClassBadRequest, false, false),
		},
		{
			name: "attachment_too_large",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorAttachmentTooLarge)},
			want: mapped(AppErrorAttachmentTooLarge, ErrorClassBadRequest, false, false),
		},
		{
			name: "invalid_params",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorInvalidParams)},
			want: mapped(AppErrorInvalidParams, ErrorClassBadRequest, false, false),
		},
		{
			name: "session_not_found",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorSessionNotFound)},
			want: mapped(AppErrorSessionNotFound, ErrorClassNotFound, false, false),
		},
		{
			name: "storage_failed",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData(AppErrorStorageFailed)},
			want: mapped(AppErrorStorageFailed, ErrorClassInternal, false, false),
		},
		{
			name: "missing appError",
			err:  &RPCError{Code: -32000, Message: "x"},
			want: mapped(AppErrorInternal, ErrorClassInternal, false, false),
		},
		{
			name: "null data",
			err:  &RPCError{Code: -32000, Message: "x", Data: json.RawMessage("null")},
			want: mapped(AppErrorInternal, ErrorClassInternal, false, false),
		},
		{
			name: "unknown appError",
			err:  &RPCError{Code: -32000, Message: "x", Data: appErrorData("future_appearance")},
			want: mapped(AppErrorInternal, ErrorClassInternal, false, false),
		},
		{
			name: "malformed data",
			err:  &RPCError{Code: -32000, Message: "x", Data: json.RawMessage(`"boom"`)},
			want: mapped(AppErrorInternal, ErrorClassInternal, false, false),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MapRPCError(tc.err); got != tc.want {
				t.Fatalf("MapRPCError = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestAsMappedError(t *testing.T) {
	rpcErr := &RPCError{Code: -32000, Message: "busy", Data: appErrorData(AppErrorConnectionBusy)}
	wrapped := fmt.Errorf("call failed: %w", rpcErr)

	var got MappedError
	if !AsMappedError(wrapped, &got) {
		t.Fatal("AsMappedError did not detect a wrapped *RPCError")
	}
	if want := mapped(AppErrorConnectionBusy, ErrorClassTransient, true, false); got != want {
		t.Fatalf("mapped = %#v, want %#v", got, want)
	}

	if AsMappedError(&LocalError{AppError: "request_timeout"}, &got) {
		t.Fatal("AsMappedError accepted a LocalError")
	}
	if AsMappedError(errors.New("plain"), &got) {
		t.Fatal("AsMappedError accepted a non-RPC error")
	}
}

func TestIsAppError(t *testing.T) {
	if !IsAppError(&LocalError{AppError: "request_timeout"}, "request_timeout") {
		t.Fatal("local appError not detected")
	}
	rpcErr := &RPCError{Code: -32000, Message: "busy", Data: appErrorData(AppErrorConnectionBusy)}
	if !IsAppError(rpcErr, AppErrorConnectionBusy) {
		t.Fatal("mapped appError not detected")
	}
	if IsAppError(rpcErr, "request_timeout") {
		t.Fatal("wrong appError matched")
	}
	if IsAppError(errors.New("plain"), AppErrorInternal) {
		t.Fatal("plain error matched")
	}
}
