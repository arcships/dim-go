package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// handshake 发送 `hello` 并校验协议版本、实例标识与服务端画像。
// 版本不符返回 VersionIncompatible（终态）；其余问题按不可重试处理。
func handshake(
	ctx context.Context,
	client *protocol.Client,
	clientName string,
) (protocol.HelloResult, error) {
	params := protocol.HelloParams{
		ProtocolVersion: protocol.ProtocolVersion,
		ClientKind:      "other",
		ClientName:      clientName,
	}
	var hello protocol.HelloResult
	if err := client.CallInto(ctx, "hello", params, &hello); err != nil {
		var rpcErr *protocol.RPCError
		if errors.As(err, &rpcErr) {
			switch rpcErr.AppError() {
			case "protocol_version", "session_protocol_version":
				return hello, failure.Wrap(failure.KindVersionIncompatible, "hello", err)
			}
		}
		return hello, failure.Wrap(failure.KindOf(err), "hello", err)
	}
	if hello.ProtocolVersion != protocol.ProtocolVersion {
		return hello, failure.New(
			failure.KindVersionIncompatible,
			"hello",
			fmt.Errorf("服务端协议版本 %d，本客户端要求 %d", hello.ProtocolVersion, protocol.ProtocolVersion),
		)
	}
	if hello.InstanceID == "" {
		return hello, failure.New(failure.KindNonRetryable, "hello", errors.New("hello 未给出 instanceId"))
	}
	switch hello.DaemonProfile.Kind {
	case "product", "mock":
	case "configured":
		if hello.DaemonProfile.ModelConfigSHA256 == "" {
			return hello, failure.New(
				failure.KindNonRetryable,
				"hello",
				errors.New("hello 的 configured 画像缺少 modelConfigSha256"),
			)
		}
	default:
		return hello, failure.New(
			failure.KindNonRetryable,
			"hello",
			fmt.Errorf("hello 的 daemonProfile 无效: %q", hello.DaemonProfile.Kind),
		)
	}
	return hello, nil
}
