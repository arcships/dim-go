package app

import (
	"errors"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/host"
)

// HostOptions 是把集成层接到宿主外壳所需的进程级参数。
type HostOptions struct {
	// DataDir 是应用数据目录，用于推导单实例端点。
	DataDir string
	// Endpoint 是单实例端点的完整路径；设置后优先于 DataDir。
	Endpoint string
	// Args 是本次启动参数；二次启动时原样转交给已有实例。
	Args []string
	// WorkingDir 是本次启动的工作目录，随转交一起送出。
	WorkingDir string
	// Version 是客户端版本，只进诊断（必须形如 x.y.z）。
	Version string
	// Diag 是白名单诊断写入端；nil 表示丢弃。
	Diag *host.Recorder
	// Timeouts 是退出各段超时；零值取宿主外壳的默认值。
	Timeouts host.Timeouts
}

// NewHost 把集成层接到宿主外壳：启动五阶段与退出四阶段都指向 App 的步骤函数，
// 并把外壳的写闸门交给 App，使未就绪、未连接与退出中的新意图在登记前被拒绝。
//
// 单实例判定、阶段推进、逐段超时与诊断都由宿主外壳驱动；本函数只做接线，
// 不复制任何阶段顺序。
func NewHost(base *App, options HostOptions) (*host.Shell, error) {
	shell, err := host.New(host.Config{
		DataDir:    options.DataDir,
		Endpoint:   options.Endpoint,
		Args:       options.Args,
		WorkingDir: options.WorkingDir,
		Version:    options.Version,
		Diag:       options.Diag,
		// 连接状态归连接领域，这里只把只读探针交给外壳。
		Connected: base.Connected,
		Startup: host.StartupSteps{
			LocalState:   base.LoadLocalState,
			Connect:      base.Connect,
			Subscribe:    base.Subscribe,
			FirstHistory: base.FirstHistory,
			AllowWrites:  base.AllowWrites,
		},
		Shutdown: host.ShutdownSteps{
			StopIntents:     base.StopIntents,
			EndInteraction:  base.EndInteraction,
			CloseConnection: base.CloseConnection,
			ReclaimService:  base.ReclaimService,
		},
		Timeouts: options.Timeouts,
	})
	if err != nil {
		return nil, err
	}
	// 更新编排在宿主外壳建好后补上退出五阶段与同一份诊断记录。
	if bindErr := base.bindUpdateHost(shell, options.Diag); bindErr != nil {
		_ = base.closeUpdate()
		return nil, bindErr
	}
	base.SetWriteGate(shell.Writable)
	return shell, nil
}

// ObserveConnection 把一次连接状态变化写进宿主外壳的诊断记录。
//
// 连接状态本身由连接领域持有，这里只转达观察值；class 为空表示正常。
func ObserveConnection(shell *host.Shell, connected bool, class host.ErrorClass) {
	if shell == nil {
		return
	}
	shell.ObserveConnection(connected, class)
}

// ClassOf 把连接领域的错误分类收敛成宿主外壳的诊断分类。
//
// 只有连接领域能确认的分类才落成具体取值；其它错误按 other 记，避免把业务错误
// 标成"不可重试"这种有行动含义的结论。
func ClassOf(err error) host.ErrorClass {
	if err == nil {
		return ""
	}
	switch failure.KindOf(err) {
	case failure.KindVersionIncompatible:
		return host.ClassVersionIncompatible
	case failure.KindConnectionMissing:
		return host.ClassConnectionMissing
	case failure.KindTimeout:
		return host.ClassTimeout
	case failure.KindRetryable:
		return host.ClassRetryable
	}
	var classified *failure.Error
	var rpcErr *protocol.RPCError
	if errors.As(err, &classified) || errors.As(err, &rpcErr) {
		return host.ClassNonRetryable
	}
	return host.ClassOther
}
