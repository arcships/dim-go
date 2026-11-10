package session

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// Call 在该会话自己的流连接上发一个请求。方法名与参数仍由 protocol 的
// wire 类型描述；这里只负责路由到正确的连接并分类错误。
func (s *Subscription) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || conn.Client.Err() != nil {
		return nil, failure.New(failure.KindRetryable, method, errors.New("会话流未连接"))
	}
	raw, err := conn.Client.Call(ctx, method, params)
	if err != nil {
		return nil, failure.Wrap(failure.KindOf(err), method, err)
	}
	return raw, nil
}

// Methods 读该会话贡献的方法清单，用于能力门禁。
func (s *Subscription) Methods(ctx context.Context) (protocol.SessionMethodsResult, error) {
	var result protocol.SessionMethodsResult
	raw, err := s.Call(ctx, protocol.MethodSessionMethods, protocol.SessionAttachParams{SessionID: s.sessionID})
	if err != nil {
		return result, err
	}
	return protocol.ParseSessionMethodsResult(raw)
}

// InputLookup 核对一次提交是否已写入；写登记用它结束「结果未知」。
func (s *Subscription) InputLookup(
	ctx context.Context,
	params protocol.InputLookupParams,
) (protocol.InputLookupResult, error) {
	var result protocol.InputLookupResult
	params.SessionID = s.sessionID
	raw, err := s.Call(ctx, protocol.MethodSessionInputLookup, params)
	if err != nil {
		return result, err
	}
	return protocol.ParseInputLookupResult(raw)
}

// PendingInputs 读未消费输入。
func (s *Subscription) PendingInputs(
	ctx context.Context,
	query protocol.PendingQuery,
) (protocol.SessionInputPendingResult, error) {
	var result protocol.SessionInputPendingResult
	raw, err := s.pendingCall(ctx, protocol.MethodSessionInputPending, query)
	if err != nil {
		return result, err
	}
	return protocol.ParseSessionInputPendingResult(raw)
}

// PendingExecTasks 读未结束的后台任务。
func (s *Subscription) PendingExecTasks(
	ctx context.Context,
	query protocol.PendingQuery,
) (protocol.SessionExecPendingResult, error) {
	var result protocol.SessionExecPendingResult
	raw, err := s.pendingCall(ctx, protocol.MethodSessionExecPending, query)
	if err != nil {
		return result, err
	}
	return protocol.ParseSessionExecPendingResult(raw)
}

// PendingQuestions 读待答提问。
func (s *Subscription) PendingQuestions(
	ctx context.Context,
	query protocol.PendingQuery,
) (protocol.SessionQuestionPendingResult, error) {
	var result protocol.SessionQuestionPendingResult
	raw, err := s.pendingCall(ctx, protocol.MethodSessionQuestionPending, query)
	if err != nil {
		return result, err
	}
	return protocol.ParseSessionQuestionPendingResult(raw)
}

// PendingCollaboration 读未结束的协作请求。
func (s *Subscription) PendingCollaboration(
	ctx context.Context,
	query protocol.PendingQuery,
) (protocol.SessionCollaborationPendingResult, error) {
	var result protocol.SessionCollaborationPendingResult
	raw, err := s.pendingCall(ctx, protocol.MethodSessionCollaborationPending, query)
	if err != nil {
		return result, err
	}
	return protocol.ParseSessionCollaborationPendingResult(raw)
}

// PutAttachment 把一段文本存成会话文件。
func (s *Subscription) PutAttachment(
	ctx context.Context,
	params protocol.AttachmentPutParams,
) (protocol.AttachmentPutResult, error) {
	var result protocol.AttachmentPutResult
	params.SessionID = s.sessionID
	raw, err := s.Call(ctx, protocol.MethodAttachmentPut, params)
	if err != nil {
		return result, err
	}
	return protocol.ParseAttachmentPutResult(raw)
}

func (s *Subscription) pendingCall(
	ctx context.Context,
	method string,
	query protocol.PendingQuery,
) (json.RawMessage, error) {
	query.SessionID = s.sessionID
	return s.Call(ctx, method, query)
}
