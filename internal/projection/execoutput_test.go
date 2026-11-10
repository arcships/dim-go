package projection

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// fakeOutputReader 按顺序交回预设的页，记下每次请求。
type fakeOutputReader struct {
	pages []ExecOutputPage
	err   error
	reqs  []ExecOutputRequest
}

func (f *fakeOutputReader) ReadExecOutput(
	_ context.Context,
	req ExecOutputRequest,
) (ExecOutputPage, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return ExecOutputPage{}, f.err
	}
	if len(f.pages) == 0 {
		return ExecOutputPage{}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

// taskFolder 造一个带未结束任务的折叠器。
func taskFolder(t *testing.T, taskID string) *Folder {
	t.Helper()
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("start1", 1, "exec", "started", `{"task_id":"`+taskID+`","description":"demo"}`),
	))
	return folder
}

// TestExecOutputPagesByBytes 验证游标按服务端给的字节数前进（不是文本长度），
// 运行中的空页不算 EOF，结束后读到空页才算 EOF。
func TestExecOutputPagesByBytes(t *testing.T) {
	folder := taskFolder(t, "task-1")
	reader := &fakeOutputReader{pages: []ExecOutputPage{
		// 文本比字节短：服务端丢掉了控制字符。
		{Text: "abc", Bytes: 5},
		// 服务端给了 nextCursor：以它为准。
		{Text: "de", Bytes: 2, NextCursor: 9},
		// 运行中的空页。
		{},
	}}
	output, err := NewExecOutput(folder, mainBranch, "task-1", reader)
	if err != nil {
		t.Fatalf("创建输出读取器: %v", err)
	}
	output.SetLimit(8)

	first, err := output.Next(context.Background())
	if err != nil {
		t.Fatalf("读第一页: %v", err)
	}
	if first.Bytes != 5 || first.Cursor != 5 || first.EOF {
		t.Fatalf("第一页的字节与游标不对: %+v", first)
	}
	if reader.reqs[0].Cursor != 0 || reader.reqs[0].Limit != 8 || reader.reqs[0].TaskID != "task-1" {
		t.Fatalf("第一页请求不对: %+v", reader.reqs[0])
	}

	second, err := output.Next(context.Background())
	if err != nil {
		t.Fatalf("读第二页: %v", err)
	}
	if second.Cursor != 9 || second.EOF {
		t.Fatalf("服务端给的 nextCursor 没被采用: %+v", second)
	}

	// 任务还在跑：空页只表示暂时没有新输出。
	third, err := output.Next(context.Background())
	if err != nil {
		t.Fatalf("读第三页: %v", err)
	}
	if third.EOF || third.Bytes != 0 || output.EOF() {
		t.Fatalf("运行中的空页不该当 EOF: %+v", third)
	}

	// 任务结束之后，再读到空页就是 EOF。
	mustNotice(t, folder, mainBranch, ev("end1", 2, "exec", "ended", `{"task_id":"task-1","status":"completed"}`))
	last, err := output.Next(context.Background())
	if err != nil {
		t.Fatalf("读最后一页: %v", err)
	}
	if !last.EOF || !output.EOF() {
		t.Fatalf("任务结束后空页应判 EOF: %+v", last)
	}
	if output.Cursor() != 9 {
		t.Fatalf("EOF 后游标不该再动: %d", output.Cursor())
	}
	// EOF 之后不再向服务端要页。
	before := len(reader.reqs)
	if _, err := output.Next(context.Background()); err != nil {
		t.Fatalf("EOF 后读取: %v", err)
	}
	if len(reader.reqs) != before {
		t.Fatalf("EOF 之后仍向服务端要了页: %d -> %d", before, len(reader.reqs))
	}
}

// TestExecOutputKeepsOnlyCursor 验证读取器不缓存整份输出：每页文本交出去后，
// 状态里只有游标。
func TestExecOutputKeepsOnlyCursor(t *testing.T) {
	folder := taskFolder(t, "task-2")
	reader := &fakeOutputReader{pages: []ExecOutputPage{
		{Text: "第一页", Bytes: 9},
		{Text: "第二页", Bytes: 9},
	}}
	output, err := NewExecOutput(folder, mainBranch, "task-2", reader)
	if err != nil {
		t.Fatalf("创建输出读取器: %v", err)
	}
	var texts []string
	for index := 0; index < 2; index++ {
		chunk, err := output.Next(context.Background())
		if err != nil {
			t.Fatalf("读第 %d 页: %v", index+1, err)
		}
		texts = append(texts, chunk.Text)
	}
	if !reflect.DeepEqual(texts, []string{"第一页", "第二页"}) {
		t.Fatalf("页文本不对: %v", texts)
	}
	if output.Cursor() != 18 {
		t.Fatalf("游标应按字节累计: %d", output.Cursor())
	}
}

// TestExecOutputLimitClamp 验证单页上限的取值规则。
func TestExecOutputLimitClamp(t *testing.T) {
	folder := taskFolder(t, "task-3")
	reader := &fakeOutputReader{}
	output, err := NewExecOutput(folder, mainBranch, "task-3", reader)
	if err != nil {
		t.Fatalf("创建输出读取器: %v", err)
	}
	output.SetLimit(0)
	if _, err := output.Next(context.Background()); err != nil {
		t.Fatalf("读取: %v", err)
	}
	if reader.reqs[0].Limit != defaultExecOutputPageBytes {
		t.Fatalf("默认上限不对: %d", reader.reqs[0].Limit)
	}
	output.SetLimit(1 << 30)
	if _, err := output.Next(context.Background()); err != nil {
		t.Fatalf("读取: %v", err)
	}
	if reader.reqs[1].Limit != maxExecOutputPageBytes {
		t.Fatalf("上限应被夹到服务端最大值: %d", reader.reqs[1].Limit)
	}
}

// TestExecOutputReportsErrors 验证构造参数与读取错误都原样带回。
func TestExecOutputReportsErrors(t *testing.T) {
	if _, err := NewExecOutput(nil, mainBranch, "t", nil); !errors.Is(err, ErrExecOutputNoReader) {
		t.Fatalf("缺读取来源应失败: %v", err)
	}
	if _, err := NewExecOutput(nil, mainBranch, "", &fakeOutputReader{}); !errors.Is(err, ErrExecOutputNoTask) {
		t.Fatalf("缺任务标识应失败: %v", err)
	}
	reader := &fakeOutputReader{err: errors.New("断开")}
	output, err := NewExecOutput(nil, mainBranch, "t", reader)
	if err != nil {
		t.Fatalf("创建输出读取器: %v", err)
	}
	if _, err := output.Next(context.Background()); err == nil {
		t.Fatalf("读取失败应原样返回")
	}
}
