// Package failure 定义连接领域的错误分类。
//
// 上层（进程框架、会话视图、写登记）只按分类决定是否重试、是否失败关闭，
// 不解析底层错误文本。这里只覆盖连接领域能确认的失败：版本不兼容、连接缺失、
// 超时、可重试、不可重试。写操作结果未知不在此列，由写登记的「在途写」处理。
package failure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"syscall"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// Kind 是一类可执行的失败结论。
type Kind string

const (
	// KindVersionIncompatible 表示服务端协议版本与本客户端不符。终态：停止重试，
	// 直到进程重启或服务被替换。
	KindVersionIncompatible Kind = "version_incompatible"
	// KindConnectionMissing 表示本地端点不存在或无人监听。可重新发现或拉起服务，
	// 但不能按普通重试盲目重发。
	KindConnectionMissing Kind = "connection_missing"
	// KindTimeout 表示请求在超时时间内没有回执。请求是否已生效未知。
	KindTimeout Kind = "timeout"
	// KindRetryable 表示连接被打断或服务暂时忙，可在恢复连接后重试。
	KindRetryable Kind = "retryable"
	// KindNonRetryable 表示重试不会得到不同结果（能力缺失、参数错误、终态）。
	KindNonRetryable Kind = "non_retryable"
)

// Error 是带分类的连接领域错误。
type Error struct {
	Kind Kind
	Op   string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	op := e.Op
	if op == "" {
		op = "appserver"
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", op, e.Kind)
	}
	return fmt.Sprintf("%s: %s: %v", op, e.Kind, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// Is 使 errors.Is(err, &failure.Error{Kind: target}) 按分类成立。
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e.Kind == other.Kind
}

// New 构造一条分类错误；kind 由调用方按上下文给出。
func New(kind Kind, op string, err error) *Error {
	return &Error{Kind: kind, Op: op, Err: err}
}

// Wrap 在保留底层错误的同时标明分类；已带分类的错误原样返回。
func Wrap(kind Kind, op string, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	return &Error{Kind: kind, Op: op, Err: err}
}

// KindOf 给出错误的分类。没有可识别特征时按不可重试处理（失败关闭）。
func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var classified *Error
	if errors.As(err, &classified) {
		return classified.Kind
	}
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		return kindForRPC(rpcErr)
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, protocol.ErrRequestTimeout):
		return KindTimeout
	case errors.Is(err, context.Canceled):
		return KindNonRetryable
	case errors.Is(err, protocol.ErrConnectionClosed),
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrUnexpectedEOF),
		errors.Is(err, net.ErrClosed):
		return KindRetryable
	case isMissingEndpoint(err):
		return KindConnectionMissing
	default:
		return KindNonRetryable
	}
}

// Retryable 表示恢复连接后重试是允许的；超时也算可重试，但调用方必须记住
// 请求可能已经生效。
func Retryable(err error) bool {
	kind := KindOf(err)
	return kind == KindRetryable || kind == KindTimeout
}

// Is 判断错误是否属于某一分类。
func Is(err error, kind Kind) bool { return KindOf(err) == kind }

// kindForRPC 把服务端稳定 appError 映射为分类。映射表与产品基线的错误边界
// 一致：能力缺失与参数错误不可重试，忙与订阅变化可重试，版本不符是终态。
func kindForRPC(err *protocol.RPCError) Kind {
	switch err.AppError() {
	case "protocol_version", "session_protocol_version":
		return KindVersionIncompatible
	case "request_timeout":
		return KindTimeout
	case "connection_closed", "connection_busy", "service_reloading", "subscription_changed":
		return KindRetryable
	}
	switch err.Code {
	case -32602, -32601: // invalid params / method not found
		return KindNonRetryable
	}
	return KindNonRetryable
}

func isMissingEndpoint(err error) bool {
	return errors.Is(err, fs.ErrNotExist) ||
		errors.Is(err, syscall.ENOENT) ||
		errors.Is(err, syscall.ECONNREFUSED)
}
