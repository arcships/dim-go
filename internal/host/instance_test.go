package host

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSecondLaunchForwardsArgsAndPrimaryKeepsRunning(t *testing.T) {
	endpoint := shortEndpoint(t)
	diag, buf := newTestDiag(t)

	primary, err := TryAcquire(InstanceConfig{Endpoint: endpoint, Diag: diag})
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	defer primary.Close()

	if _, err := TryAcquire(InstanceConfig{Endpoint: endpoint}); !errors.Is(err, ErrInstanceRunning) {
		t.Fatalf("第二次启动应返回 ErrInstanceRunning，实际 %v", err)
	}

	secret := "sk-live-0123456789abcdef"
	err = Forward(InstanceConfig{Endpoint: endpoint}, Activation{
		Args:       []string{"--open", secret},
		WorkingDir: "/tmp/work",
	})
	if err != nil {
		t.Fatalf("Forward: %v", err)
	}
	activation := waitActivation(t, primary.Activations())
	if len(activation.Args) != 2 || activation.Args[0] != "--open" || activation.Args[1] != secret {
		t.Fatalf("转交参数 = %v", activation.Args)
	}
	if activation.WorkingDir != "/tmp/work" {
		t.Fatalf("转交工作目录 = %q", activation.WorkingDir)
	}
	if activation.At.IsZero() {
		t.Fatal("转交时间未记录")
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("转交参数内容进入诊断: %s", buf.String())
	}

	if err := primary.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(endpoint); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("关闭后端点仍存在: %v", err)
	}
	restarted, err := TryAcquire(InstanceConfig{Endpoint: endpoint})
	if err != nil {
		t.Fatalf("关闭后应能重新占用端点: %v", err)
	}
	_ = restarted.Close()
}

func TestForwardWithoutPrimaryFails(t *testing.T) {
	endpoint := shortEndpoint(t)
	err := Forward(InstanceConfig{Endpoint: endpoint, ForwardTimeout: 100 * time.Millisecond}, Activation{})
	if err == nil {
		t.Fatal("没有主实例时转交应失败")
	}
	if errors.Is(err, ErrInstanceRunning) {
		t.Fatal("转交失败不应报成已有实例")
	}
}

func TestActivationOverflowDropsWithoutLoggingContent(t *testing.T) {
	endpoint := shortEndpoint(t)
	diag, buf := newTestDiag(t)
	instance, err := TryAcquire(InstanceConfig{Endpoint: endpoint, Diag: diag, Backlog: 1})
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	defer instance.Close()

	if err := Forward(InstanceConfig{Endpoint: endpoint}, Activation{Args: []string{"first"}}); err != nil {
		t.Fatalf("第一次转交: %v", err)
	}
	secret := "sk-live-deadbeef"
	if err := Forward(InstanceConfig{Endpoint: endpoint}, Activation{Args: []string{secret}}); err != nil {
		t.Fatalf("第二次转交: %v", err)
	}

	activation := waitActivation(t, instance.Activations())
	if len(activation.Args) != 1 || activation.Args[0] != "first" {
		t.Fatalf("队列里的接管 = %v，期望第一次", activation.Args)
	}
	records := recordsOf(decodeRecords(t, buf.String()), EventActivationDropped)
	if len(records) != 1 {
		t.Fatalf("丢弃记录条数 = %d，期望 1", len(records))
	}
	if records[0]["args"] != float64(1) {
		t.Fatalf("丢弃记录只应含参数个数，实际 %v", records[0])
	}
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("被丢弃的参数内容进入诊断: %s", buf.String())
	}
}

func TestEndpointPath(t *testing.T) {
	if got := EndpointPath(""); got != "" {
		t.Fatalf("空数据目录 = %q，期望空串", got)
	}
	got := EndpointPath("/tmp/dim")
	want := filepath.Join("/tmp/dim", instanceDirName, instanceFileName)
	if got != want {
		t.Fatalf("EndpointPath = %q，期望 %q", got, want)
	}
	if _, err := (InstanceConfig{}).endpointPath(); err == nil {
		t.Fatal("未配置端点与数据目录时应报错")
	}
	if _, err := (InstanceConfig{DataDir: "/tmp/dim"}).endpointPath(); err != nil {
		t.Fatalf("由数据目录推导端点失败: %v", err)
	}
}

func TestInstanceBacklogDefault(t *testing.T) {
	endpoint := shortEndpoint(t)
	instance, err := TryAcquire(InstanceConfig{Endpoint: endpoint})
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	defer instance.Close()
	if cap(instance.Activations()) != defaultBacklog {
		t.Fatalf("默认队列长度 = %d，期望 %d", cap(instance.Activations()), defaultBacklog)
	}
}
