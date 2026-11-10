package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// instanceDirName 是实例端点在数据目录下的子目录名。
	instanceDirName = "host"
	// activationAction 是二次启动转交的动作名。
	activationAction = "activate"
	// maxActivationBytes 是单次转交报文的上限，防止无界读取。
	maxActivationBytes = 1 << 20
	// activationDeadline 是主实例读取一次转交报文的期限，避免慢连接拖住退出。
	activationDeadline = time.Second
	// defaultBacklog 是待处理接管队列的默认长度。
	defaultBacklog = 8
	// defaultForwardTimeout 是二次启动转交的默认超时。
	defaultForwardTimeout = time.Second
	// defaultProbeTimeout 是判定端点是否被占用时的默认拨号超时。
	defaultProbeTimeout = 300 * time.Millisecond
)

// ErrInstanceRunning 表示本机已有主实例在运行，本次启动应转交参数后退出。
var ErrInstanceRunning = errors.New("host: 已有实例在运行")

// Activation 是一次二次启动转交过来的启动参数。
type Activation struct {
	// Args 是转交的启动参数，原样保存，不解释语义。
	Args []string
	// WorkingDir 是二次启动时的工作目录。
	WorkingDir string
	// At 是主实例收到转交的时间。
	At time.Time
}

// InstanceConfig 是单实例端点的配置。
type InstanceConfig struct {
	// Endpoint 是完整端点路径；为空时由 DataDir 推导。
	Endpoint string
	// DataDir 是应用数据目录，用于推导默认端点。
	DataDir string
	// Backlog 是待处理接管队列长度，小于等于 0 时取默认值。
	Backlog int
	// ForwardTimeout 是转交超时，小于等于 0 时取默认值。
	ForwardTimeout time.Duration
	// ProbeTimeout 是判定端点是否被占用的拨号超时，小于等于 0 时取默认值。
	ProbeTimeout time.Duration
	// Diag 是诊断写入端，可为 nil。
	Diag *Recorder
}

// EndpointPath 返回数据目录下的默认实例端点路径；dataDir 为空时返回空串。
func EndpointPath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, instanceDirName, instanceFileName)
}

func (c InstanceConfig) endpointPath() (string, error) {
	if c.Endpoint != "" {
		return c.Endpoint, nil
	}
	path := EndpointPath(c.DataDir)
	if path == "" {
		return "", errors.New("host: 未配置实例端点或数据目录")
	}
	return path, nil
}

func (c InstanceConfig) backlog() int {
	if c.Backlog > 0 {
		return c.Backlog
	}
	return defaultBacklog
}

func (c InstanceConfig) forwardTimeout() time.Duration {
	if c.ForwardTimeout > 0 {
		return c.ForwardTimeout
	}
	return defaultForwardTimeout
}

func (c InstanceConfig) probeTimeout() time.Duration {
	if c.ProbeTimeout > 0 {
		return c.ProbeTimeout
	}
	return defaultProbeTimeout
}

// Instance 是主实例持有的端点与接管队列。
type Instance struct {
	endpoint    string
	ln          net.Listener
	diag        *Recorder
	activations chan Activation
	done        chan struct{}
	timeout     time.Duration

	closeOnce sync.Once
	closeErr  error
	wg        sync.WaitGroup
}

// TryAcquire 尝试成为主实例。端点已被其他实例占用时返回 ErrInstanceRunning；
// 被强杀后残留的端点按"无人监听"清理后重新占用，不依赖任何锁文件或状态文件。
func TryAcquire(cfg InstanceConfig) (*Instance, error) {
	path, err := cfg.endpointPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("host: 创建实例端点目录失败: %w", err)
	}
	ln, stale, err := acquireLocal(path, cfg.probeTimeout())
	if err != nil {
		return nil, err
	}
	instance := newInstance(ln, path, cfg)
	if stale {
		cfg.Diag.Log(LevelInfo, EventInstanceStaleRemoved)
	}
	return instance, nil
}

// Forward 把本次启动参数转交给已有主实例。成功后调用方应直接退出。
func Forward(cfg InstanceConfig, activation Activation) error {
	path, err := cfg.endpointPath()
	if err != nil {
		return err
	}
	timeout := cfg.forwardTimeout()
	conn, err := dialLocal(path, timeout)
	if err != nil {
		return fmt.Errorf("host: 无法连接已有实例: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	request := activationRequest{
		Action:     activationAction,
		Args:       append([]string(nil), activation.Args...),
		WorkingDir: activation.WorkingDir,
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return fmt.Errorf("host: 转交启动参数失败: %w", err)
	}
	var response activationResponse
	if err := json.NewDecoder(io.LimitReader(conn, maxActivationBytes)).Decode(&response); err != nil {
		return fmt.Errorf("host: 转交启动参数后未收到回执: %w", err)
	}
	if !response.OK {
		return errors.New("host: 已有实例拒绝了本次转交")
	}
	return nil
}

func newInstance(ln net.Listener, path string, cfg InstanceConfig) *Instance {
	instance := &Instance{
		endpoint:    path,
		ln:          ln,
		diag:        cfg.Diag,
		activations: make(chan Activation, cfg.backlog()),
		done:        make(chan struct{}),
		timeout:     activationDeadline,
	}
	instance.wg.Add(1)
	go instance.serve()
	return instance
}

// Endpoint 返回本实例占用的端点。
func (i *Instance) Endpoint() string { return i.endpoint }

// Activations 返回接管队列；主实例关闭后该通道关闭。
func (i *Instance) Activations() <-chan Activation { return i.activations }

// Close 停止监听、移除端点并关闭接管队列。重复调用返回同一结果。
func (i *Instance) Close() error {
	if i == nil {
		return nil
	}
	i.closeOnce.Do(func() {
		close(i.done)
		// 先释放端点名字再停监听：否则在两步之间，二次启动会把仍可连上但即将关闭的
		// 端点判为残留，接管后的新端点又被本进程删除。
		_ = os.Remove(i.endpoint)
		i.closeErr = i.ln.Close()
		i.wg.Wait()
		close(i.activations)
	})
	return i.closeErr
}

func (i *Instance) serve() {
	defer i.wg.Done()
	for {
		conn, err := i.ln.Accept()
		if err != nil {
			return
		}
		i.wg.Add(1)
		go func() {
			defer i.wg.Done()
			i.handle(conn)
		}()
	}
}

func (i *Instance) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(i.timeout))

	var request activationRequest
	decoder := json.NewDecoder(io.LimitReader(conn, maxActivationBytes))
	if err := decoder.Decode(&request); err != nil {
		return
	}
	if request.Action != activationAction {
		_ = json.NewEncoder(conn).Encode(activationResponse{OK: false})
		return
	}

	activation := Activation{
		Args:       append([]string(nil), request.Args...),
		WorkingDir: request.WorkingDir,
		At:         time.Now(),
	}
	delivered := false
	select {
	case i.activations <- activation:
		delivered = true
	case <-i.done:
		return
	default:
		// 队列已满：只记参数个数，不记参数内容。
		i.diag.Log(LevelWarn, EventActivationDropped, ArgsField(len(activation.Args)))
	}
	_ = json.NewEncoder(conn).Encode(activationResponse{OK: true, Delivered: delivered})
}

type activationRequest struct {
	Action     string   `json:"action"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
}

type activationResponse struct {
	OK        bool `json:"ok"`
	Delivered bool `json:"delivered"`
}
