package protocol

import (
	"encoding/json"
	"errors"
)

// MethodSessionMethods 读会话贡献方法清单，用来判断某项能力是否存在。
const MethodSessionMethods = "session.methods"

// SessionMethod 是一条会话方法声明。
type SessionMethod struct {
	Name     string `json:"name"`
	Mutating bool   `json:"mutating"`
}

// SessionMethodsResult 是 `session.methods` 回执。
type SessionMethodsResult struct {
	Methods []SessionMethod `json:"methods"`
}

// ParseSessionMethodsResult 解码并校验方法清单；未知字段被忽略。
func ParseSessionMethodsResult(raw json.RawMessage) (SessionMethodsResult, error) {
	var result SessionMethodsResult
	if err := decodeJSON(raw, &result, "session.methods"); err != nil {
		return result, err
	}
	if result.Methods == nil {
		return result, errors.New("session.methods is missing methods")
	}
	for _, method := range result.Methods {
		if method.Name == "" {
			return result, errors.New("session.methods has an empty method name")
		}
	}
	return result, nil
}

// Has 判断清单里是否声明了某个方法。
func (r SessionMethodsResult) Has(name string) bool {
	for _, method := range r.Methods {
		if method.Name == name {
			return true
		}
	}
	return false
}
