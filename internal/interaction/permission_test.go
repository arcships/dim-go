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

// TestPermissionHasNoDefault 核对：没有取值时不猜默认值，也不往提交意图里加字段。
func TestPermissionHasNoDefault(t *testing.T) {
	arbiter, _ := newTestArbiter(t, testSession)
	if tier, ok := arbiter.PermissionTier(); ok || tier != "" {
		t.Fatalf("初始不应有档位: %q %v", tier, ok)
	}
	intent := submitIntent(testSession, json.RawMessage(`{"text":"你好"}`))
	got, err := arbiter.WithPermission(intent)
	if err != nil {
		t.Fatalf("无档位不应报错: %v", err)
	}
	if string(got.Params) != string(intent.Params) {
		t.Fatalf("无档位时不应改参数: %s", got.Params)
	}
}

// TestPermissionIsPassedThrough 核对：档位保存后显式透传给本会话的提交意图，已有字段保留。
func TestPermissionIsPassedThrough(t *testing.T) {
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
		t.Fatalf("透传档位: %v", err)
	}
	params := decodeParams(t, got.Params)
	if params["permission"] != string(PermissionWorkspaceWrite) {
		t.Fatalf("档位没有透传: %v", params)
	}
	if params["text"] != "你好" || params["mode"] != "queue" {
		t.Fatalf("原有参数被改坏: %v", params)
	}
	if string(intent.Params) != string(original) {
		t.Fatalf("入参不应被就地修改: %s", intent.Params)
	}

	// 空参数也能合入。
	empty, err := arbiter.WithPermission(submitIntent(testSession, nil))
	if err != nil {
		t.Fatalf("空参数透传: %v", err)
	}
	emptyParams := decodeParams(t, empty.Params)
	if emptyParams["permission"] != string(PermissionWorkspaceWrite) {
		t.Fatalf("空参数没有合入档位: %v", emptyParams)
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

// TestPermissionRejectsEmptyAndBadParams 核对：空档位与非对象参数被拒绝。
func TestPermissionRejectsEmptyAndBadParams(t *testing.T) {
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

	for _, raw := range []json.RawMessage{
		json.RawMessage(`[1,2]`),
		json.RawMessage(`"文本"`),
		json.RawMessage(`3`),
	} {
		intent := submitIntent(testSession, raw)
		if _, err := arbiter.WithPermission(intent); !errors.Is(err, ErrPermissionParams) {
			t.Fatalf("非对象参数应被拒: %s (%v)", raw, err)
		}
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
}
