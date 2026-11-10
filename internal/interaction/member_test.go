package interaction

import (
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// TestStopMemberOnlyStopsMember 核对：成员停止只发带成员标识的停止意图，不动主线与待决集合。
func TestStopMemberOnlyStopsMember(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	arbiter.Apply(approvalFact("a1"))
	before := arbiter.Counts()

	view, err := arbiter.StopMember(MemberRef{MemberID: "m1", BranchID: "b1"})
	if err != nil {
		t.Fatalf("停止成员: %v", err)
	}
	if view.Kind != write.KindStop {
		t.Fatalf("意图类型不对: %s", view.Kind)
	}

	stops := writer.intentsOf(write.KindStop)
	if len(stops) != 1 {
		t.Fatalf("应只发一条停止意图: %d", len(stops))
	}
	stop := stops[0]
	if stop.SessionID != testSession || stop.BranchID != "b1" {
		t.Fatalf("停止目标不对: %+v", stop)
	}
	params := decodeParams(t, stop.Params)
	if params["memberId"] != "m1" {
		t.Fatalf("停止意图缺少成员标识: %v", params)
	}
	if !strings.Contains(stop.Key, "member-stop") || !strings.Contains(stop.Key, "m1") {
		t.Fatalf("停止意图的幂等键应限定到成员: %s", stop.Key)
	}

	if got := arbiter.Counts(); got != before {
		t.Fatalf("成员停止不应改变待决集合: %+v", got)
	}
	if current, ok := arbiter.Current(); !ok || current.RequestID != "a1" {
		t.Fatalf("成员停止不应影响当前待决项: %+v ok=%v", current, ok)
	}
	if len(writer.kinds()) != 1 {
		t.Fatalf("除成员停止外不应写其它意图: %v", writer.kinds())
	}
}

// TestStopMemberIsIdempotent 核对：重复停止同一成员使用同一幂等键。
func TestStopMemberIsIdempotent(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	if _, err := arbiter.StopMember(MemberRef{MemberID: "m1", BranchID: "b1"}); err != nil {
		t.Fatalf("停止成员: %v", err)
	}
	if _, err := arbiter.StopMember(MemberRef{MemberID: "m1", BranchID: "b1"}); err != nil {
		t.Fatalf("重复停止成员: %v", err)
	}
	stops := writer.intentsOf(write.KindStop)
	if len(stops) != 2 || stops[0].Key != stops[1].Key {
		t.Fatalf("重复停止应复用同一幂等键: %+v", stops)
	}
}

// TestStopMemberRequiresMember 核对：缺成员标识时不发任何意图。
func TestStopMemberRequiresMember(t *testing.T) {
	arbiter, writer := newTestArbiter(t, testSession)
	if _, err := arbiter.StopMember(MemberRef{BranchID: "b1"}); !errors.Is(err, ErrInvalidMember) {
		t.Fatalf("缺成员标识应被拒: %v", err)
	}
	if writer.count() != 0 {
		t.Fatalf("拒绝时不应写意图: %v", writer.kinds())
	}
}
