package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

type stubCollaboration struct {
	snapshot CollaborationPolicySnapshot
	err      error
	calls    int
}

func (s *stubCollaboration) ReadCollaborationPolicy(context.Context) (CollaborationPolicySnapshot, error) {
	s.calls++
	if s.err != nil {
		return CollaborationPolicySnapshot{}, s.err
	}
	return s.snapshot, nil
}

func collaborationFixture(revision string, mode CollaborationPolicyMode) CollaborationPolicySnapshot {
	return CollaborationPolicySnapshot{Revision: revision, Policy: CollaborationModelPolicy{Mode: mode}}
}

func TestRefreshCollaborationPolicy(t *testing.T) {
	reader := &stubCollaboration{snapshot: collaborationFixture("r1", CollaborationPolicyAgent)}
	manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: reader})
	snapshot, err := manager.RefreshCollaborationPolicy(context.Background())
	if err != nil {
		t.Fatalf("读取协作策略: %v", err)
	}
	if snapshot.Revision != "r1" || snapshot.Policy.Mode != CollaborationPolicyAgent {
		t.Fatalf("协作策略不符：%+v", snapshot)
	}
	cached, ok := manager.CollaborationPolicy()
	if !ok || cached.Revision != "r1" {
		t.Fatalf("协作策略缓存不符：%+v", cached)
	}

	bad := []CollaborationPolicySnapshot{
		{Revision: "", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent}},
		{Revision: "r1", Policy: CollaborationModelPolicy{Mode: "unknown"}},
		{Revision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyFixed}},
		{Revision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent, ModelID: "m"}},
	}
	for _, snapshot := range bad {
		manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: &stubCollaboration{snapshot: snapshot}})
		if _, err := manager.RefreshCollaborationPolicy(context.Background()); !errors.Is(err, ErrMalformedCollaboration) {
			t.Fatalf("不合法策略应拒绝：%+v -> %v", snapshot, err)
		}
	}
}

func TestSetCollaborationPolicyWritesOnceAndRefreshes(t *testing.T) {
	reader := &stubCollaboration{snapshot: collaborationFixture("r1", CollaborationPolicyAgent)}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		reader.snapshot = CollaborationPolicySnapshot{
			Revision: "r2",
			Policy:   CollaborationModelPolicy{Mode: CollaborationPolicyFixed, ModelProviderID: "p1", ModelID: "m1"},
		}
		return write.Reply{Result: json.RawMessage(`{"revision":"r2"}`)}, nil
	}}
	manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: reader, Writer: newTestWriter(t, sender)})
	if _, err := manager.RefreshCollaborationPolicy(context.Background()); err != nil {
		t.Fatalf("预读策略: %v", err)
	}

	snapshot, err := manager.SetCollaborationPolicy(context.Background(), CollaborationPolicyRequest{
		ExpectedRevision: "r1",
		Policy:           CollaborationModelPolicy{Mode: CollaborationPolicyFixed, ModelProviderID: "p1", ModelID: "m1"},
	})
	if err != nil {
		t.Fatalf("写入协作策略: %v", err)
	}
	if snapshot.Revision != "r2" || snapshot.Policy.Mode != CollaborationPolicyFixed || snapshot.Policy.ModelID != "m1" {
		t.Fatalf("写入后策略不符：%+v", snapshot)
	}
	requests := sender.snapshot()
	if len(requests) != 1 || requests[0].Kind != write.KindManagementWrite {
		t.Fatalf("协作策略应经管理写入一次：%+v", requests)
	}
	params := decodeParams[collaborationPolicyParams](t, requests[0].Params)
	if params.Op != "collaboration.set_model_policy" || params.ExpectedRevision != "r1" ||
		params.Policy.Mode != CollaborationPolicyFixed || params.Policy.Model == nil ||
		params.Policy.Model.ProviderID != "p1" || params.Policy.Model.ModelID != "m1" {
		t.Fatalf("协作策略参数不符：%+v", params)
	}
	if reader.calls != 2 {
		t.Fatalf("写入成功后应重读一次：%d", reader.calls)
	}
}

func TestSetCollaborationPolicyConflicts(t *testing.T) {
	t.Run("本地冲突不发请求", func(t *testing.T) {
		reader := &stubCollaboration{snapshot: collaborationFixture("r2", CollaborationPolicyAgent)}
		sender := &senderFunc{}
		manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: reader, Writer: newTestWriter(t, sender)})
		if _, err := manager.RefreshCollaborationPolicy(context.Background()); err != nil {
			t.Fatalf("预读策略: %v", err)
		}
		if _, err := manager.SetCollaborationPolicy(context.Background(), CollaborationPolicyRequest{
			ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent},
		}); !errors.Is(err, ErrCollaborationConflict) {
			t.Fatalf("本地版本不符应冲突：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("本地冲突不应发请求")
		}
	})

	t.Run("服务端冲突只读一次", func(t *testing.T) {
		reader := &stubCollaboration{snapshot: collaborationFixture("r1", CollaborationPolicyAgent)}
		sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
			reader.snapshot = collaborationFixture("r2", CollaborationPolicyAgent)
			return write.Reply{}, errors.New("版本已变")
		}}
		manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: reader, Writer: newTestWriter(t, sender)})
		if _, err := manager.RefreshCollaborationPolicy(context.Background()); err != nil {
			t.Fatalf("预读策略: %v", err)
		}
		if _, err := manager.SetCollaborationPolicy(context.Background(), CollaborationPolicyRequest{
			ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent},
		}); !errors.Is(err, ErrCollaborationConflict) {
			t.Fatalf("服务端版本冲突应分类：%v", err)
		}
		if len(sender.snapshot()) != 1 || reader.calls != 2 {
			t.Fatalf("冲突不应重发且只读一次：writes=%d reads=%d", len(sender.snapshot()), reader.calls)
		}
	})
}

func TestSetCollaborationPolicyFailureAndValidation(t *testing.T) {
	reader := &stubCollaboration{snapshot: collaborationFixture("r1", CollaborationPolicyAgent)}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("策略被拒绝")
	}}
	manager := mustNewManager(t, Config{Features: allFeatures(), Collaboration: reader, Writer: newTestWriter(t, sender)})
	if _, err := manager.RefreshCollaborationPolicy(context.Background()); err != nil {
		t.Fatalf("预读策略: %v", err)
	}
	_, err := manager.SetCollaborationPolicy(context.Background(), CollaborationPolicyRequest{
		ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyInherit},
	})
	if !errors.Is(err, ErrCollaborationWriteFailed) || !strings.Contains(err.Error(), "策略被拒绝") {
		t.Fatalf("写入失败应保留原因：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("写入失败不应重发")
	}

	for _, request := range []CollaborationPolicyRequest{
		{Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent}},
		{ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: "unknown"}},
		{ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyFixed}},
		{ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent, ModelProviderID: "p"}},
	} {
		if _, err := manager.SetCollaborationPolicy(context.Background(), request); !errors.Is(err, ErrInvalidCollaborationRequest) {
			t.Fatalf("不合法策略请求应拒绝：%+v -> %v", request, err)
		}
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("不合法请求不应再发写入")
	}
}

func TestCollaborationPolicyCapabilities(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshCollaborationPolicy(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺读取能力应分类：%v", err)
	}
	if _, err := manager.SetCollaborationPolicy(context.Background(), CollaborationPolicyRequest{
		ExpectedRevision: "r1", Policy: CollaborationModelPolicy{Mode: CollaborationPolicyAgent},
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺写入能力应分类：%v", err)
	}
}
