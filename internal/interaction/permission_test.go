package interaction

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// submitIntent 构造一条属于指定会话的提交意图。
func submitIntent(sessionID string, params json.RawMessage) write.Intent {
	return write.Intent{Kind: write.KindSubmitInput, SessionID: sessionID, Params: params}
}

// TestPermissionHasNoDefault 核对：没有取值时不猜默认值，提交被拒绝。
func TestPermissionHasNoDefault(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if tier, ok := arbiter.PermissionTier(); ok || tier != "" {
		t.Fatalf("初始不应有档位: %q %v", tier, ok)
	}
	intent := submitIntent(testSession, json.RawMessage(`{"text":"你好"}`))
	if _, err := arbiter.WithPermission(intent); !errors.Is(err, ErrPermissionUnset) {
		t.Fatalf("无档位应拒绝提交: %v", err)
	}
}

// TestPermissionKeepsSubmitParams 核对：协议没有提交 permission 字段；档位由会话
// 分支策略携带，WithPermission 只校验不污染提交参数。
func TestPermissionKeepsSubmitParams(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if err := arbiter.SetPermissionTier(PermissionWorkspaceWrite); err != nil {
		t.Fatalf("设置档位: %v", err)
	}
	if tier, ok := arbiter.PermissionTier(); !ok || tier != PermissionWorkspaceWrite {
		t.Fatalf("档位取值不对: %q %v", tier, ok)
	}

	original := json.RawMessage(`{"text":"你好","mode":"queue"}`)
	intent := submitIntent(testSession, original)
	got, err := arbiter.WithPermission(intent)
	if err != nil {
		t.Fatalf("有档位时提交应通过: %v", err)
	}
	if string(got.Params) != string(original) {
		t.Fatalf("提交参数不应被改写: %s", got.Params)
	}
	params := decodeParams(t, got.Params)
	if _, exists := params["permission"]; exists {
		t.Fatalf("提交不应携带协议没有的 permission 字段: %v", params)
	}
	if params["text"] != "你好" || params["mode"] != "queue" {
		t.Fatalf("原有参数被改坏: %v", params)
	}
}

// TestPermissionValuesAreOpaque 核对：档位取值原样保存，不改写成本地常量。
func TestPermissionValuesAreOpaque(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if err := arbiter.SetPermissionTier("workspace_auto_v2"); err != nil {
		t.Fatalf("设置档位: %v", err)
	}
	if tier, _ := arbiter.PermissionTier(); tier != "workspace_auto_v2" {
		t.Fatalf("取值被改写: %q", tier)
	}
}

// TestPermissionRejectsEmptyTier 核对：空档位被拒绝，且不改已有取值。
func TestPermissionRejectsEmptyTier(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if err := arbiter.SetPermissionTier(PermissionFullAccess); err != nil {
		t.Fatalf("设置档位: %v", err)
	}
	if err := arbiter.SetPermissionTier(""); !errors.Is(err, ErrPermissionTierEmpty) {
		t.Fatalf("空档位应被拒: %v", err)
	}
	if tier, _ := arbiter.PermissionTier(); tier != PermissionFullAccess {
		t.Fatalf("拒绝后不应改档位: %q", tier)
	}

}

// TestPermissionScopeMatchesSession 核对：档位只透传给提交意图，且只作用于本会话。
func TestPermissionScopeMatchesSession(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if err := arbiter.SetPermissionTier(PermissionFullAccess); err != nil {
		t.Fatalf("设置档位: %v", err)
	}

	approval := write.Intent{Kind: write.KindApprovalResponse, SessionID: testSession}
	if _, err := arbiter.WithPermission(approval); !errors.Is(err, ErrNotSubmitIntent) {
		t.Fatalf("非提交意图应被拒: %v", err)
	}
	foreign := submitIntent("s2", nil)
	if _, err := arbiter.WithPermission(foreign); !errors.Is(err, ErrPermissionSession) {
		t.Fatalf("别的会话的提交意图应被拒: %v", err)
	}
	emptySession := submitIntent("", nil)
	if _, err := arbiter.WithPermission(emptySession); !errors.Is(err, ErrPermissionSession) {
		t.Fatalf("缺会话的提交意图应被拒: %v", err)
	}
}

// TestPermissionIsPerSession 核对：每个打开的会话各有一份档位，互不影响。
func TestPermissionIsPerSession(t *testing.T) {
	first, _ := newTestArbiter(t, "s1")
	second, _ := newTestArbiter(t, "s2")

	if err := first.SetPermissionTier(PermissionFullAccess); err != nil {
		t.Fatalf("设置第一个会话档位: %v", err)
	}
	if err := second.SetPermissionTier(PermissionWorkspaceWrite); err != nil {
		t.Fatalf("设置第二个会话档位: %v", err)
	}
	if tier, _ := first.PermissionTier(); tier != PermissionFullAccess {
		t.Fatalf("第一个会话档位不对: %q", tier)
	}
	if tier, _ := second.PermissionTier(); tier != PermissionWorkspaceWrite {
		t.Fatalf("第二个会话档位不对: %q", tier)
	}

	first.ClearPermissionTier()
	if _, ok := first.PermissionTier(); ok {
		t.Fatal("第一个会话清空后不应有档位")
	}
	if tier, ok := second.PermissionTier(); !ok || tier != PermissionWorkspaceWrite {
		t.Fatalf("第二个会话档位不应受影响: %q %v", tier, ok)
	}
}

// TestPermissionClear 核对：清空后回到没有值。
func TestPermissionClear(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if err := arbiter.SetPermissionTier(PermissionFullAccess); err != nil {
		t.Fatalf("设置档位: %v", err)
	}
	arbiter.ClearPermissionTier()
	if tier, ok := arbiter.PermissionTier(); ok || tier != "" {
		t.Fatalf("清空后不应有值: %q %v", tier, ok)
	}
	if _, err := arbiter.WithPermission(submitIntent(testSession, nil)); !errors.Is(err, ErrPermissionUnset) {
		t.Fatalf("清空后提交应被拒: %v", err)
	}
}
