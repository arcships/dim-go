package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

type stubAgentAdmin struct {
	agents          []AgentProfile
	agentsErr       error
	listCalls       int
	current         SessionAgent
	currentErr      error
	currentCalls    int
	supported       bool
	supportsGet     bool
	supportErr      error
	supportCalls    int
	supportGetCalls int
}

func (s *stubAgentAdmin) ListAgents(context.Context) ([]AgentProfile, error) {
	s.listCalls++
	if s.agentsErr != nil {
		return nil, s.agentsErr
	}
	return s.agents, nil
}

func (s *stubAgentAdmin) ReadSessionAgent(context.Context, string, string) (SessionAgent, error) {
	s.currentCalls++
	if s.currentErr != nil {
		return SessionAgent{}, s.currentErr
	}
	return s.current, nil
}

func (s *stubAgentAdmin) SupportsSessionAgentGet(context.Context, string) (bool, error) {
	s.supportGetCalls++
	if s.supportErr != nil {
		return false, s.supportErr
	}
	return s.supportsGet, nil
}

func (s *stubAgentAdmin) SupportsSessionAgentUpdate(context.Context, string) (bool, error) {
	s.supportCalls++
	if s.supportErr != nil {
		return false, s.supportErr
	}
	return s.supported, nil
}

func TestRefreshAgentsAndValidation(t *testing.T) {
	admin := &stubAgentAdmin{agents: []AgentProfile{
		{ID: "a1", Name: "Reviewer", Sessions: 1},
		{ID: "a2", Name: "Default", Default: true},
	}}
	manager := mustNewManager(t, Config{Features: allFeatures(), Agents: admin})
	agents, err := manager.RefreshAgents(context.Background())
	if err != nil {
		t.Fatalf("读取 agent 清单: %v", err)
	}
	if len(agents) != 2 || agents[0].ID != "a1" || agents[1].Name != "Default" {
		t.Fatalf("agent 清单不符：%+v", agents)
	}
	agents[0].Name = "changed"
	cached, ok := manager.Agents()
	if !ok || cached[0].Name != "Reviewer" {
		t.Fatalf("agent 缓存被调用方改动：%+v", cached)
	}

	for _, bad := range [][]AgentProfile{
		{{Name: "missing id"}},
		{{ID: "a", Name: "x"}, {ID: "a", Name: "y"}},
		{{ID: "a", Name: ""}},
		{{ID: "a", Name: "x", Sessions: -1}},
	} {
		manager := mustNewManager(t, Config{Features: allFeatures(), Agents: &stubAgentAdmin{agents: bad}})
		if _, err := manager.RefreshAgents(context.Background()); !errors.Is(err, ErrMalformedAgent) {
			t.Fatalf("不合法 agent 清单应拒绝：%+v -> %v", bad, err)
		}
	}
}

func TestReadSessionAgent(t *testing.T) {
	admin := &stubAgentAdmin{current: SessionAgent{AgentID: "a1"}, supportsGet: true}
	manager := mustNewManager(t, Config{Features: allFeatures(), Agents: admin})
	agent, err := manager.ReadSessionAgent(context.Background(), "s1", "main")
	if err != nil {
		t.Fatalf("读取会话 agent: %v", err)
	}
	if agent.SessionID != "s1" || agent.BranchID != "main" || agent.AgentID != "a1" {
		t.Fatalf("会话 agent 不符：%+v", agent)
	}
	if _, err := manager.ReadSessionAgent(context.Background(), "", "main"); !errors.Is(err, ErrInvalidAgentRequest) {
		t.Fatalf("缺会话应拒绝：%v", err)
	}
	if _, err := manager.ReadSessionAgent(context.Background(), "s1", ""); !errors.Is(err, ErrInvalidAgentRequest) {
		t.Fatalf("缺分支应拒绝：%v", err)
	}

	missing := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := missing.ReadSessionAgent(context.Background(), "s1", "main"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺读取接口应分类：%v", err)
	}
	noGet := mustNewManager(t, Config{
		Features: allFeatures(),
		Agents:   &stubAgentAdmin{current: SessionAgent{AgentID: "a1"}},
	})
	if _, err := noGet.ReadSessionAgent(context.Background(), "s1", "main"); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺 get 方法应分类：%v", err)
	}
}

func TestUpdateSessionAgentWritesOnceAndReadsBack(t *testing.T) {
	admin := &stubAgentAdmin{supported: true, supportsGet: true}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		admin.current = SessionAgent{AgentID: "a1"}
		return write.Reply{Result: json.RawMessage(`{"outcome":"adopted"}`)}, nil
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender),
	})
	result, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
		SessionID: "s1", BranchID: "main", AgentID: "a1",
	})
	if err != nil {
		t.Fatalf("更新会话 agent: %v", err)
	}
	if result.AgentID != "a1" || !result.Changed || result.SessionID != "s1" {
		t.Fatalf("更新结果不符：%+v", result)
	}
	if len(sender.snapshot()) != 1 || sender.snapshot()[0].Kind != write.KindManagementWrite {
		t.Fatalf("agent 更新应经管理写入一次：%+v", sender.snapshot())
	}
	params := decodeParams[sessionAgentUpdateParams](t, sender.snapshot()[0].Params)
	if params.Op != "session.agent.update" || params.SessionID != "s1" || params.BranchID != "main" ||
		params.AgentID != "a1" || params.ExpectedAgentID != "" {
		t.Fatalf("agent 更新参数不符：%+v", params)
	}
	if admin.currentCalls != 2 || admin.supportCalls != 1 {
		t.Fatalf("读取/门禁次数不符：reads=%d supports=%d", admin.currentCalls, admin.supportCalls)
	}
}

func TestUpdateSessionAgentConflictAndCapability(t *testing.T) {
	t.Run("本地当前 agent 不符不发请求", func(t *testing.T) {
		admin := &stubAgentAdmin{current: SessionAgent{AgentID: "a2"}, supported: true, supportsGet: true}
		sender := &senderFunc{}
		manager := mustNewManager(t, Config{Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender)})
		if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
			SessionID: "s1", BranchID: "main", AgentID: "a1", ExpectedAgentID: "a1",
		}); !errors.Is(err, ErrAgentConflict) {
			t.Fatalf("当前 agent 不符应冲突：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("冲突不应发请求")
		}
	})

	t.Run("缺更新方法零请求", func(t *testing.T) {
		admin := &stubAgentAdmin{supported: false, supportsGet: true}
		sender := &senderFunc{}
		manager := mustNewManager(t, Config{Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender)})
		if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
			SessionID: "s1", BranchID: "main", AgentID: "a1",
		}); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("缺更新方法应分类：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("缺方法不应发请求")
		}
		if admin.currentCalls != 0 {
			t.Fatal("缺更新方法不应先读当前 agent")
		}
	})

	t.Run("方法表读取失败零请求", func(t *testing.T) {
		admin := &stubAgentAdmin{supportErr: errors.New("方法表不可读"), supportsGet: true}
		sender := &senderFunc{}
		manager := mustNewManager(t, Config{Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender)})
		if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
			SessionID: "s1", BranchID: "main", AgentID: "a1",
		}); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("方法表失败应分类：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("方法表失败不应发请求")
		}
	})

	t.Run("缺 get 能力零请求", func(t *testing.T) {
		admin := &stubAgentAdmin{supported: true, supportsGet: false}
		sender := &senderFunc{}
		manager := mustNewManager(t, Config{
			Features: stubFeatures{FeatureSessionAgentUpdate: true},
			Agents:   admin,
			Writer:   newTestWriter(t, sender),
		})
		if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
			SessionID: "s1", BranchID: "main", AgentID: "a1",
		}); !errors.Is(err, ErrCapabilityMissing) {
			t.Fatalf("缺 get 能力应分类：%v", err)
		}
		if len(sender.snapshot()) != 0 {
			t.Fatal("缺 get 能力不应发请求")
		}
	})
}

func TestUpdateSessionAgentFailureKeepsReasonAndReadsOnce(t *testing.T) {
	admin := &stubAgentAdmin{supported: true, supportsGet: true}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("档案不可用")
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender),
	})
	_, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
		SessionID: "s1", BranchID: "main", AgentID: "a1",
	})
	if !errors.Is(err, ErrAgentWriteFailed) || !strings.Contains(err.Error(), "档案不可用") {
		t.Fatalf("写入失败应保留原因：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("写入失败不应重发")
	}
	if admin.currentCalls != 2 {
		t.Fatalf("失败后应只读一次当前 agent：%d", admin.currentCalls)
	}
}

func TestUpdateSessionAgentServerChangedBecomesConflict(t *testing.T) {
	admin := &stubAgentAdmin{supported: true, supportsGet: true}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		admin.current = SessionAgent{AgentID: "a2"}
		return write.Reply{}, errors.New("版本冲突")
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender),
	})
	if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
		SessionID: "s1", BranchID: "main", AgentID: "a1",
	}); !errors.Is(err, ErrAgentConflict) {
		t.Fatalf("服务端当前 agent 已变应冲突：%v", err)
	}
}

func TestUpdateSessionAgentReadbackMismatch(t *testing.T) {
	admin := &stubAgentAdmin{supported: true, supportsGet: true}
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		admin.current = SessionAgent{AgentID: "a2"}
		return write.Reply{Result: json.RawMessage(`{"outcome":"adopted"}`)}, nil
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(), Agents: admin, Writer: newTestWriter(t, sender),
	})
	if _, err := manager.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
		SessionID: "s1", BranchID: "main", AgentID: "a1",
	}); !errors.Is(err, ErrAgentWriteFailed) {
		t.Fatalf("回读不一致应报写入失败：%v", err)
	}
}

func TestAgentCapabilitiesAndWriter(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshAgents(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺 agent 读取接口应分类：%v", err)
	}
	admin := &stubAgentAdmin{supported: true, supportsGet: true}
	noWriter := mustNewManager(t, Config{Features: allFeatures(), Agents: admin})
	if _, err := noWriter.UpdateSessionAgent(context.Background(), SessionAgentUpdateRequest{
		SessionID: "s1", BranchID: "main", AgentID: "a1",
	}); !errors.Is(err, ErrWriteUnavailable) {
		t.Fatalf("缺写登记应分类：%v", err)
	}
}
