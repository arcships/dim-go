package protocol_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

func mustRaw(t *testing.T, text string) json.RawMessage {
	t.Helper()
	if !json.Valid([]byte(text)) {
		t.Fatalf("test payload is not JSON: %s", text)
	}
	return json.RawMessage(text)
}

func TestDecodeApprovalNotices(t *testing.T) {
	t.Run("request tolerates unknown fields", func(t *testing.T) {
		raw := mustRaw(t, `{
			"sessionId":"s-1","branchId":"main","requestId":"req-1",
			"source":{"kind":"tool","turn_id":"t-1","request_id":"r-1","call_id":"c-1","future":1},
			"call":{"name":"exec","input":{"command":"ls"},"extra":true},
			"reason":"needs approval","future":1
		}`)
		notice, err := protocol.ParseApprovalRequestNotice(raw)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if notice.SessionID != "s-1" || notice.RequestID != "req-1" || notice.Source.Kind != "tool" {
			t.Fatalf("unexpected notice: %+v", notice)
		}
		if notice.Source.CallID != "c-1" || notice.Call.Name != "exec" {
			t.Fatalf("nested fields: %+v", notice)
		}
		if string(notice.Call.Input) != `{"command":"ls"}` {
			t.Fatalf("input not preserved: %s", notice.Call.Input)
		}
	})

	t.Run("request rejects a missing requestId", func(t *testing.T) {
		raw := mustRaw(t, `{
			"sessionId":"s-1","branchId":"main",
			"source":{"kind":"tool"},"call":{"name":"exec"},"reason":"r"
		}`)
		if _, err := protocol.ParseApprovalRequestNotice(raw); err == nil {
			t.Fatal("want an error for a missing requestId")
		}
	})

	t.Run("cancelled tolerates unknown fields and rejects a missing branchId", func(t *testing.T) {
		notice, err := protocol.ParseApprovalCancelledNotice(mustRaw(t,
			`{"sessionId":"s","branchId":"main","requestId":"r","reason":"stopped","future":true}`))
		if err != nil || notice.RequestID != "r" || notice.Reason != "stopped" {
			t.Fatalf("decode: %+v, %v", notice, err)
		}
		if _, err := protocol.ParseApprovalCancelledNotice(mustRaw(t,
			`{"sessionId":"s","requestId":"r","reason":"stopped"}`)); err == nil {
			t.Fatal("want an error for a missing branchId")
		}
	})

	t.Run("resolved tolerates unknown fields and rejects an unknown decision", func(t *testing.T) {
		notice, err := protocol.ParseApprovalResolvedNotice(mustRaw(t,
			`{"sessionId":"s","branchId":"main","requestId":"r","decision":"allow","future":1}`))
		if err != nil || notice.Decision != protocol.ApprovalDecisionAllow {
			t.Fatalf("decode: %+v, %v", notice, err)
		}
		if _, err := protocol.ParseApprovalResolvedNotice(mustRaw(t,
			`{"sessionId":"s","branchId":"main","requestId":"r","decision":"maybe"}`)); err == nil {
			t.Fatal("want an error for an unknown decision")
		}
	})
}

func TestDecodeSessionNoticeTitle(t *testing.T) {
	raw := mustRaw(t, `{
		"sessionId":"s","source":"SessionStore","kind":"title",
		"data":{"title":"Hello","titleSource":"auto","titleRevision":2,"future":1},
		"future":2
	}`)
	notice, err := protocol.ParseSessionNotice(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !notice.IsTitle() {
		t.Fatal("notice must be recognized as a title change")
	}
	title, err := notice.ParseTitle()
	if err != nil {
		t.Fatalf("parse title: %v", err)
	}
	if title.SessionID != "s" || title.Title == nil || *title.Title != "Hello" {
		t.Fatalf("unexpected title: %+v", title)
	}
	if title.TitleSource == nil || *title.TitleSource != "auto" || title.TitleRevision != 2 {
		t.Fatalf("unexpected title fields: %+v", title)
	}

	if _, err := protocol.ParseSessionNotice(mustRaw(t, `{"sessionId":"s","kind":"title","data":{}}`)); err == nil {
		t.Fatal("want an error for a missing source")
	}
	if _, err := (protocol.SessionNotice{
		SessionID: "s", Source: protocol.NoticeSourceSessionStore, Kind: protocol.NoticeKindTitle,
		Data: mustRaw(t, `{"title":"x","titleSource":"manual"}`),
	}).ParseTitle(); err == nil {
		t.Fatal("want an error when titleRevision is missing")
	}
}

func TestDecodeInputLookup(t *testing.T) {
	accepted, err := protocol.ParseInputLookupResult(mustRaw(t, `{
		"status":"accepted",
		"event":{"id":"e","sequence":3,"create_at":1,"type":"input","name":"message","data":{},"future":1},
		"future":2
	}`))
	if err != nil || accepted.Event == nil || accepted.Event.Sequence != 3 {
		t.Fatalf("decode accepted: %+v, %v", accepted, err)
	}
	unknown, err := protocol.ParseInputLookupResult(mustRaw(t, `{"status":"unknown","future":1}`))
	if err != nil || unknown.Status != protocol.InputLookupUnknown || unknown.Event != nil {
		t.Fatalf("decode unknown: %+v, %v", unknown, err)
	}
	if _, err := protocol.ParseInputLookupResult(mustRaw(t, `{"status":"accepted"}`)); err == nil {
		t.Fatal("want an error for accepted without an event")
	}
}

func TestDecodePendingInputs(t *testing.T) {
	result, err := protocol.ParseSessionInputPendingResult(mustRaw(t, `{
		"branchId":"main","through":5,"next":{"binding":"b","keys":[]},"future":1,
		"inputs":[{
			"inputId":"i-1","sequence":2,"mode":"queue","applied":true,"future":2,
			"user":{"content":[{"type":"text","text":"hi"}]}
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.HasNext() || len(result.Inputs) != 1 {
		t.Fatalf("unexpected page: %+v", result)
	}
	input := result.Inputs[0]
	if input.InputID != "i-1" || input.Mode != protocol.InputModeQueue || !input.Applied || input.User == nil {
		t.Fatalf("unexpected input: %+v", input)
	}
	if _, err := protocol.ParseSessionInputPendingResult(mustRaw(t, `{
		"branchId":"main","through":1,"inputs":[{"sequence":2,"mode":"queue","applied":false}]
	}`)); err == nil {
		t.Fatal("want an error for an input without inputId")
	}
	if _, err := protocol.ParseSessionInputPendingResult(mustRaw(t,
		`{"branchId":"main","through":1}`)); err == nil {
		t.Fatal("want an error for a missing inputs list")
	}
}

func TestDecodePendingExecTasks(t *testing.T) {
	result, err := protocol.ParseSessionExecPendingResult(mustRaw(t, `{
		"branchId":"main","through":7,"next":null,"future":1,
		"tasks":[{
			"eventId":"e","sequence":2,"startedAt":10,"taskId":"t","turnId":"u",
			"requestId":"r","callId":"c","description":"build","future":2
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.HasNext() || result.Tasks[0].TaskID != "t" || result.Tasks[0].StartedAt != 10 {
		t.Fatalf("unexpected page: %+v", result)
	}
	if _, err := protocol.ParseSessionExecPendingResult(mustRaw(t, `{
		"branchId":"main","through":1,"tasks":[{"eventId":"e","sequence":1}]
	}`)); err == nil {
		t.Fatal("want an error for a task without taskId")
	}
}

func TestDecodePendingQuestions(t *testing.T) {
	result, err := protocol.ParseSessionQuestionPendingResult(mustRaw(t, `{
		"branchId":"main","through":9,"next":null,"future":1,
		"batches":[{
			"eventId":"e","sequence":4,"requestId":"r","future":2,
			"calls":[{"id":"c","name":"ask_user","input":{"questions":[]},"future":3}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Batches) != 1 || result.Batches[0].Calls[0].Name != "ask_user" {
		t.Fatalf("unexpected page: %+v", result)
	}
	if _, err := protocol.ParseSessionQuestionPendingResult(mustRaw(t, `{
		"branchId":"main","through":1,"batches":[{"eventId":"e","sequence":2,"calls":[]}]
	}`)); err == nil {
		t.Fatal("want an error for a batch without requestId")
	}
}

func TestDecodePendingCollaboration(t *testing.T) {
	result, err := protocol.ParseSessionCollaborationPendingResult(mustRaw(t, `{
		"branchId":"main","through":3,"next":null,"future":1,
		"requests":[{
			"eventId":"e","sequence":4,"startedAt":10,"future":2,
			"request":{
				"turn_id":"t","request_id":"r","call_id":"c","prompt":"do it","future":3,
				"target":{"kind":"member","agent_id":"a","session_id":"s","future":4}
			},
			"delivery":{"status":"ok","agent_id":"a","session_id":"s","input_event_id":"i","future":5}
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	request := result.Requests[0]
	if request.Request.Target.Kind != "member" || request.Delivery == nil {
		t.Fatalf("unexpected page: %+v", result)
	}
	if request.Delivery.Status != protocol.CollaborationDeliveryOK || request.Delivery.InputEventID != "i" {
		t.Fatalf("unexpected delivery: %+v", request.Delivery)
	}
	if _, err := protocol.ParseSessionCollaborationPendingResult(mustRaw(t, `{
		"branchId":"main","through":1,
		"requests":[{"eventId":"e","sequence":2,"request":{"target":{"kind":"agent"}}}]
	}`)); err == nil {
		t.Fatal("want an error for a request without turn_id")
	}
}

func TestDecodeAttachmentPut(t *testing.T) {
	result, err := protocol.ParseAttachmentPutResult(mustRaw(t,
		`{"attachmentId":"a-1","path":"/tmp/file.txt","bytes":12,"future":1}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.AttachmentID != "a-1" || result.Bytes != 12 || !strings.HasSuffix(result.Path, "file.txt") {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := protocol.ParseAttachmentPutResult(mustRaw(t, `{"path":"/tmp/x","bytes":1}`)); err == nil {
		t.Fatal("want an error for a missing attachmentId")
	}
	if _, err := protocol.ParseAttachmentPutResult(mustRaw(t,
		`{"attachmentId":"a","path":"/tmp/x","bytes":-1}`)); err == nil {
		t.Fatal("want an error for negative bytes")
	}
}

func TestDecodeSessionMethods(t *testing.T) {
	result, err := protocol.ParseSessionMethodsResult(mustRaw(t, `{
		"methods":[{"name":"attachment.put","mutating":true,"future":1}],"future":2
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Has(protocol.MethodAttachmentPut) {
		t.Fatal("method list must report attachment.put")
	}
	if _, err := protocol.ParseSessionMethodsResult(mustRaw(t, `{}`)); err == nil {
		t.Fatal("want an error for a missing methods list")
	}
}
