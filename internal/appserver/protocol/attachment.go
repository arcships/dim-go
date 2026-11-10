package protocol

import (
	"encoding/json"
	"errors"
)

// MethodAttachmentPut 是把大段粘贴存成会话文件的方法。
const MethodAttachmentPut = "attachment.put"

// AttachmentPutParams 是 `attachment.put` 参数；Name 省略时服务端用 paste.txt。
type AttachmentPutParams struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name,omitempty"`
	Text      string `json:"text"`
}

// AttachmentPutResult 是 `attachment.put` 回执，AttachmentID 即会话文件标识。
type AttachmentPutResult struct {
	AttachmentID string `json:"attachmentId"`
	Path         string `json:"path"`
	Bytes        int64  `json:"bytes"`
}

// ParseAttachmentPutResult 解码并校验一次附件写入；未知字段被忽略。
func ParseAttachmentPutResult(raw json.RawMessage) (AttachmentPutResult, error) {
	var result AttachmentPutResult
	if err := decodeJSON(raw, &result, "attachment.put"); err != nil {
		return result, err
	}
	switch {
	case result.AttachmentID == "":
		return result, errors.New("attachment.put is missing attachmentId")
	case result.Path == "":
		return result, errors.New("attachment.put is missing path")
	case result.Bytes < 0:
		return result, errors.New("attachment.put has a negative bytes count")
	}
	return result, nil
}
