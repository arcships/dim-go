package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// UI 宿主方法名。它们只对本机连接开放（hello 用 ui.leaf.2 标记叶子语义）。
const (
	MethodUIOpen     = "ui.open"
	MethodUIAttach   = "ui.attach"
	MethodUIClose    = "ui.close"
	MethodUIStatus   = "ui.status"
	MethodUIShutdown = "ui.shutdown"
	MethodUIReplace  = "ui.replace"
)

// UiKindTUI 是 ui.open 目前接受的叶子类型；普通连接注册可用任意可打印 kind。
const UiKindTUI = "tui"

// UI 注册状态。
const (
	UiStateOnline    = "online"
	UiStateStarting  = "starting"
	UiStateReplacing = "replacing"
)

// UIOpenParams 是 `ui.open` 参数：在 address 上打开一个 leaf 会话。
// Peer 是 leaf 的 rutis 标识（`ui-` + 小写字母/数字/-），即返回的 uiId。
type UIOpenParams struct {
	SessionID          string  `json:"sessionId"`
	Kind               string  `json:"kind"`
	Peer               string  `json:"peer"`
	Address            string  `json:"address"`
	Replace            *string `json:"replace,omitempty"`
	ExpectedGeneration *int64  `json:"expectedGeneration,omitempty"`
}

// UIOpened 是 `ui.open` 回执：token 只在叶子首次 attach 时使用。
type UIOpened struct {
	UIID       string `json:"uiId"`
	Generation int64  `json:"generation"`
	Token      string `json:"token"`
}

// ParseUIOpened 解码并校验一次叶子打开；未知字段被忽略。
func ParseUIOpened(raw json.RawMessage) (UIOpened, error) {
	var opened UIOpened
	if err := decodeJSON(raw, &opened, "ui.open"); err != nil {
		return opened, err
	}
	if opened.UIID == "" || opened.Token == "" || opened.Generation < 1 {
		return opened, errors.New("ui.open is missing uiId, token or generation")
	}
	return opened, nil
}

// UIAttachParams 是 `ui.attach` 参数：普通界面不带 uiId/token；叶子带
// `ui.open` 回执里的两者。
type UIAttachParams struct {
	SessionID string  `json:"sessionId"`
	Kind      string  `json:"kind"`
	UIID      *string `json:"uiId,omitempty"`
	Token     *string `json:"token,omitempty"`
}

// UIAttached 是 `ui.attach` 回执。
type UIAttached struct {
	UIID       string `json:"uiId"`
	Generation int64  `json:"generation"`
}

// ParseUIAttached 解码并校验一次界面注册；未知字段被忽略。
func ParseUIAttached(raw json.RawMessage) (UIAttached, error) {
	var attached UIAttached
	if err := decodeJSON(raw, &attached, "ui.attach"); err != nil {
		return attached, err
	}
	if attached.UIID == "" || attached.Generation < 1 {
		return attached, errors.New("ui.attach is missing uiId or generation")
	}
	return attached, nil
}

// UICloseParams 是 `ui.close` 参数：移除注册并关闭叶子的会话。
type UICloseParams struct {
	UIID   string  `json:"uiId"`
	Reason *string `json:"reason,omitempty"`
}

// UIReplaceParams 是 `ui.replace` 参数：连接注册的替换，旧界面随后关闭。
type UIReplaceParams struct {
	UIID       string `json:"uiId"`
	SessionID  string `json:"sessionId"`
	Kind       string `json:"kind"`
	Generation *int64 `json:"generation,omitempty"`
}

// UIShutdownParams 是 `ui.shutdown` 请求参数；服务器据此向该界面的连接
// 发一条 advisory 通知。
type UIShutdownParams struct {
	UIID   string  `json:"uiId"`
	Reason *string `json:"reason,omitempty"`
}

// UIShutdownNotice 是 `ui.shutdown` 通知：只提醒界面被关闭，不是强行终止。
type UIShutdownNotice struct {
	UIID   string `json:"uiId"`
	Reason string `json:"reason"`
}

// ParseUIShutdownNotice 解码并校验一条界面关闭通知；未知字段被忽略。
func ParseUIShutdownNotice(raw json.RawMessage) (UIShutdownNotice, error) {
	var notice UIShutdownNotice
	if err := decodeJSON(raw, &notice, "ui.shutdown"); err != nil {
		return notice, err
	}
	if notice.UIID == "" || notice.Reason == "" {
		return notice, errors.New("ui.shutdown is missing uiId or reason")
	}
	return notice, nil
}

// UIInstance 是 `ui.status` 列表里的一条界面。
type UIInstance struct {
	UIID       string `json:"uiId"`
	SessionID  string `json:"sessionId"`
	Kind       string `json:"kind"`
	SinceMS    int64  `json:"sinceMs"`
	Generation int64  `json:"generation"`
	State      string `json:"state"`
}

// UIStatusResult 是 `ui.status` 回执。
type UIStatusResult struct {
	Instances      []UIInstance `json:"instances"`
	WebConnections int          `json:"webConnections"`
}

// ParseUIStatusResult 解码并校验界面清单；未知字段被忽略。
func ParseUIStatusResult(raw json.RawMessage) (UIStatusResult, error) {
	var result UIStatusResult
	if err := decodeJSON(raw, &result, "ui.status"); err != nil {
		return result, err
	}
	if result.Instances == nil {
		return result, errors.New("ui.status is missing instances")
	}
	for index, instance := range result.Instances {
		if instance.UIID == "" || instance.SessionID == "" || instance.Kind == "" {
			return result, fmt.Errorf("ui.status instances[%d] is missing identity", index)
		}
		switch instance.State {
		case UiStateOnline, UiStateStarting, UiStateReplacing:
		default:
			return result, fmt.Errorf("ui.status instances[%d] has an invalid state %q", index, instance.State)
		}
	}
	return result, nil
}
