package host

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer 是并发安全的写入端，测试读取时不会与后台接管协程争用。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestDiag(t *testing.T) (*Recorder, *syncBuffer) {
	t.Helper()
	buf := &syncBuffer{}
	return NewRecorder(buf), buf
}

func decodeRecords(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("诊断行不是合法 JSON: %v\n%s", err, line)
		}
		records = append(records, rec)
	}
	return records
}

// recordAllowedKey 判断记录里的键是否属于固定记录头或白名单字段。
func recordAllowedKey(key string) bool {
	switch key {
	case "time", "level", "event":
		return true
	default:
		return fieldKeys[key]
	}
}

func recordsOf(records []map[string]any, event Event) []map[string]any {
	var matched []map[string]any
	for _, rec := range records {
		if rec["event"] == string(event) {
			matched = append(matched, rec)
		}
	}
	return matched
}

// shortEndpoint 在系统临时目录下建一个短路径端点，避开 Unix 套接字路径长度上限。
func shortEndpoint(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "dhm-")
	if err != nil {
		t.Fatalf("建临时目录失败: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, instanceFileName)
}

// classifiedError 是自报分类的测试错误，模拟连接领域包装后的错误。
type classifiedError struct {
	msg   string
	class ErrorClass
}

func (e classifiedError) Error() string { return e.msg }

func (e classifiedError) ErrorClass() string { return string(e.class) }

// waitActivation 等待一次接管，超时即失败。
func waitActivation(t *testing.T, ch <-chan Activation) Activation {
	t.Helper()
	select {
	case activation, ok := <-ch:
		if !ok {
			t.Fatal("接管队列已关闭")
		}
		return activation
	case <-time.After(2 * time.Second):
		t.Fatal("等待接管超时")
		return Activation{}
	}
}
