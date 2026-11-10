package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// JSONRPCVersion 是唯一接受的 JSON-RPC 版本。
const JSONRPCVersion = "2.0"

// ID 是 JSON-RPC 请求标识在 wire 上的原始文本（JSON 数字或 JSON 字符串）。
// 用原始文本作为关联键，数字 1 与字符串 "1" 不会被混为一谈。
type ID string

// MarshalJSON 按原始文本写出，避免把数字标识写成字符串。
func (id ID) MarshalJSON() ([]byte, error) {
	if id == "" {
		return []byte("null"), nil
	}
	return []byte(id), nil
}

// UnmarshalJSON 原样保留标识文本，由 Valid 校验。
func (id *ID) UnmarshalJSON(raw []byte) error {
	*id = ID(bytes.TrimSpace(raw))
	return nil
}

// Valid 判断标识是否是非负整数字面量或 JSON 字符串。
func (id ID) Valid() bool {
	if id == "" {
		return false
	}
	if id[0] == '"' {
		var s string
		return json.Unmarshal([]byte(id), &s) == nil
	}
	text := string(id)
	if text[0] == '-' {
		text = text[1:]
	}
	if text == "" {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

// MessageKind 是一条入站消息的类别。
type MessageKind int

const (
	// KindNotification 是通知；未知方法原样保留。
	KindNotification MessageKind = iota + 1
	// KindResult 是成功回执。
	KindResult
	// KindError 是失败回执。
	KindError
)

// Message 是一条已解析的入站 JSON-RPC 消息。字段按类别使用：
// 通知有 Method/Params；回执有 ID，且 Result 或 Error 二选一。
type Message struct {
	Kind   MessageKind
	ID     ID
	Method string
	Params json.RawMessage
	Result json.RawMessage
	Error  *RPCError
}

// RPCError 是 JSON-RPC 错误对象。data.appError 是服务端稳定错误码。
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error 实现 error。
func (e *RPCError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if app := e.AppError(); app != "" {
		return fmt.Sprintf("rpc error %d (%s): %s", e.Code, app, e.Message)
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// AppError 返回 data.appError；没有时为空。
func (e *RPCError) AppError() string {
	if e == nil || len(e.Data) == 0 {
		return ""
	}
	var data struct {
		AppError string `json:"appError"`
	}
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return ""
	}
	return data.AppError
}

// ParseMessage 校验并解析一条 wire 消息。未知通知方法不报错，按 Method 原样保留。
func ParseMessage(frame []byte) (Message, error) {
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  *string         `json:"method"`
		Params  json.RawMessage `json:"params"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(frame, &envelope); err != nil {
		return Message{}, fmt.Errorf("parse json-rpc message: %w", err)
	}
	if envelope.JSONRPC != JSONRPCVersion {
		return Message{}, errors.New("frame is not a JSON-RPC 2.0 message")
	}
	if envelope.Method != nil {
		if len(envelope.ID) != 0 {
			return Message{}, errors.New("notification must not carry an id")
		}
		return Message{
			Kind:   KindNotification,
			Method: *envelope.Method,
			Params: envelope.Params,
		}, nil
	}
	if len(envelope.ID) == 0 {
		return Message{}, errors.New("reply has no id")
	}
	id := ID(bytes.TrimSpace(envelope.ID))
	if string(id) != "null" && !id.Valid() {
		return Message{}, errors.New("reply has an invalid id")
	}
	hasResult := len(envelope.Result) != 0
	hasError := len(envelope.Error) != 0
	if hasResult == hasError {
		return Message{}, errors.New("reply must carry exactly one of result or error")
	}
	if hasError {
		rpcErr, err := parseRPCError(envelope.Error)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindError, ID: id, Error: rpcErr}, nil
	}
	if string(id) == "null" {
		return Message{}, errors.New("successful reply has a null id")
	}
	return Message{Kind: KindResult, ID: id, Result: envelope.Result}, nil
}

func parseRPCError(raw json.RawMessage) (*RPCError, error) {
	var wire struct {
		Code    *int            `json:"code"`
		Message *string         `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, fmt.Errorf("reply has an invalid error object: %w", err)
	}
	if wire.Code == nil || wire.Message == nil {
		return nil, errors.New("reply has an invalid error object")
	}
	if len(wire.Data) != 0 && string(wire.Data) != "null" {
		var data map[string]json.RawMessage
		if err := json.Unmarshal(wire.Data, &data); err != nil {
			return nil, errors.New("reply has an invalid error data object")
		}
		if app, ok := data["appError"]; ok {
			var name string
			if err := json.Unmarshal(app, &name); err != nil {
				return nil, errors.New("reply has an invalid appError")
			}
		}
	}
	return &RPCError{Code: *wire.Code, Message: *wire.Message, Data: wire.Data}, nil
}

// Request 是发往服务端的 JSON-RPC 请求。
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      ID              `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
