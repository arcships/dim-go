package extensions

import (
	"context"
	"testing"
)

// problemReader 是同时实现清单读取与无效条目读取的读方。
type problemReader struct {
	servers    []ToolServer
	problems   []ToolServerProblem
	problemErr error
}

func (p *problemReader) ReadToolServers(context.Context) ([]ToolServer, error) {
	return p.servers, nil
}

func (p *problemReader) ProbeToolServer(context.Context, string) (ToolServerProbe, error) {
	return ToolServerProbe{}, ErrCapabilityMissing
}

func (p *problemReader) PreviewToolServerImport(context.Context, string) (ImportPreview, error) {
	return ImportPreview{}, ErrCapabilityMissing
}

func (p *problemReader) ReadToolServerProblems(context.Context) ([]ToolServerProblem, error) {
	if p.problemErr != nil {
		return nil, p.problemErr
	}
	return p.problems, nil
}

// featureSet 是最小的 hello 能力声明。
type featureSet map[string]bool

func (f featureSet) HasFeature(name string) bool { return f[name] }

// TestToolServerProblemsRead 断言无效条目随清单一起读、按副本返回，且问题读取失败
// 不清空上一次确认的问题。
func TestToolServerProblemsRead(t *testing.T) {
	reader := &problemReader{
		servers: []ToolServer{{ID: "srv-1", Name: "srv-1", Transport: ToolTransportStdio}},
		problems: []ToolServerProblem{{
			Scope: "user", ServerID: "broken", Message: "缺少 command",
		}},
	}
	service, err := New(Config{ToolServers: reader, Logf: t.Logf})
	if err != nil {
		t.Fatalf("创建服务：%v", err)
	}
	defer service.Close()

	if _, err := service.ListToolServers(context.Background()); err != nil {
		t.Fatalf("读取清单：%v", err)
	}
	problems := service.ToolServerProblems()
	if len(problems) != 1 || problems[0].ServerID != "broken" ||
		problems[0].Message != "缺少 command" {
		t.Fatalf("问题 = %+v", problems)
	}
	// 返回的是副本：改它不影响服务里的读数。
	problems[0].Message = "改掉"
	if got := service.ToolServerProblems(); got[0].Message != "缺少 command" {
		t.Fatalf("返回的不是副本：%+v", got)
	}

	reader.problemErr = context.DeadlineExceeded
	if _, err := service.ListToolServers(context.Background()); err != nil {
		t.Fatalf("第二次读取清单：%v", err)
	}
	if got := service.ToolServerProblems(); len(got) != 1 {
		t.Fatalf("问题读取失败清空了上一次读数：%+v", got)
	}
}

// TestSupportsFeature 断言能力声明决定入口：声明了为真，未声明为假，没有能力
// 来源时按「不知道」返回真（交给调用结果给原因）。
func TestSupportsFeature(t *testing.T) {
	service, err := New(Config{Features: featureSet{"mcp.remove": true}, Logf: t.Logf})
	if err != nil {
		t.Fatalf("创建服务：%v", err)
	}
	defer service.Close()
	if !service.SupportsFeature("mcp.remove") {
		t.Fatal("声明了 mcp.remove 却报不支持")
	}
	if service.SupportsFeature("mcp.save") {
		t.Fatal("没有声明 mcp.save 却报支持")
	}

	unknown, err := New(Config{Logf: t.Logf})
	if err != nil {
		t.Fatalf("创建服务：%v", err)
	}
	defer unknown.Close()
	if !unknown.SupportsFeature("mcp.save") {
		t.Fatal("没有能力声明时应按不知道处理，不置灰入口")
	}
}
