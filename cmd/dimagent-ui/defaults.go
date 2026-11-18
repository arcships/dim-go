package main

// 本文件只放启动默认值：客户端数据目录、App Server 的发现顺序、客户端启动日志。
//
// 三条约定：
//   - 默认值只有一处：不带参数启动也能起来（数据目录落产品目录，服务端按顺序找）。
//   - 发现顺序固定：-server → 环境变量 → 可执行文件同目录 → PATH → 已有端点。
//     每一档都记进日志与「找不到」的原因里，界面与日志能对上。
//   - 不读 App Server 自己的文件：这里只看端点是否存在，不碰它的数据库与会话文件。

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// envDataDir 覆盖默认数据目录（安装包/多实例调试用）。
	envDataDir = "DIM_DATA_DIR"
	// envAppServer 指向 App Server 可执行文件。
	envAppServer = "DIM_APP_SERVER"
	// legacyEnvAppServer 是旧名；仍按同一优先级接受，方便既有脚本。
	legacyEnvAppServer = "DIM_APP_SERVER_BIN"
	// serverName 是 App Server 的可执行文件名（同目录与 PATH 都按它找）。
	serverName = "dim-app-server"
	// logDirName 是客户端日志子目录；日志与数据同处一棵树，便于整目录清理。
	logDirName = "logs"
	// logFileName 是默认日志文件名。
	logFileName = "dimagent.log"
	// endpointDirName 是 App Server 运行时目录（含 daemon.sock），与连接领域一致。
	endpointDirName = "rust"
)

// resolveDataDir 给出客户端数据目录：参数 → DIM_DATA_DIR → 平台产品目录。
//
// 平台产品目录在 macOS 上是 ~/Library/Application Support/DimAgent，其他平台用
// 用户配置目录下的 DimAgent。
func resolveDataDir(flagValue string) (string, error) {
	if v := strings.TrimSpace(flagValue); v != "" {
		return expandHome(v), nil
	}
	if v := strings.TrimSpace(os.Getenv(envDataDir)); v != "" {
		return expandHome(v), nil
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("取用户主目录: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "DimAgent"), nil
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "DimAgent"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("取用户主目录: %w", err)
	}
	return filepath.Join(home, ".dimagent"), nil
}

// expandHome 把开头的 ~ 展开成用户主目录；其他形式原样返回。
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// endpointPath 是已有服务端点（Unix socket）的路径；与连接领域的推导一致。
func endpointPath(dataDir string) string {
	return filepath.Join(dataDir, endpointDirName, "daemon.sock")
}

// serverPlan 是一次发现的结果。
type serverPlan struct {
	// Path 是命中的 App Server 可执行文件；空表示不拉起，直接用已有端点。
	Path string
	// Source 说明命中来源，只进日志与界面标题，不参与连接判断。
	Source string
	// Endpoint 是已有服务端点的路径（Path 为空时才有意义）。
	Endpoint string
	// Tried 按顺序记下试过的位置，找不到时用它给出可读原因。
	Tried []string
	// Explicit 表示这次查找来自显式指定（参数或环境变量）：不可用时立即失败，
	// 不退回自动发现。
	Explicit bool
}

// describe 给出一行说明，用于日志。
func (p serverPlan) describe() string {
	switch {
	case p.Path != "":
		return fmt.Sprintf("%s（来源：%s）", p.Path, p.Source)
	case p.Source != "":
		return fmt.Sprintf("%s（%s）", p.Endpoint, p.Source)
	default:
		return "未找到"
	}
}

// findServer 按固定顺序找 App Server；都找不到时返回带「试过哪些位置」的错误。
//
// 顺序：-server → DIM_APP_SERVER（兼容旧名）→ 可执行文件同目录 → PATH → 已有端点。
// 显式给出（参数或环境变量）但不可用时立即失败：用户指了地方，就不该悄悄换成别的。
func findServer(explicit, dataDir string) (serverPlan, error) {
	plan := serverPlan{Endpoint: endpointPath(dataDir)}

	if v := strings.TrimSpace(explicit); v != "" {
		return resolveExplicit(plan, "参数 -server", v)
	}
	for _, name := range []string{envAppServer, legacyEnvAppServer} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return resolveExplicit(plan, "环境变量 "+name, v)
		}
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), serverName)
		if ok, reason := executable(candidate); ok {
			plan.Path, plan.Source = candidate, "可执行文件同目录"
			return plan, nil
		} else if reason != "" {
			plan.Tried = append(plan.Tried, candidate+"："+reason)
		} else {
			plan.Tried = append(plan.Tried, candidate)
		}
	}

	if found, err := exec.LookPath(serverName); err == nil {
		plan.Path, plan.Source = found, "PATH"
		return plan, nil
	}
	plan.Tried = append(plan.Tried, "PATH 里的 "+serverName)

	if _, err := os.Stat(plan.Endpoint); err == nil {
		plan.Source = "已有服务端点"
		return plan, nil
	}
	plan.Tried = append(plan.Tried, plan.Endpoint)

	return plan, fmt.Errorf("按顺序都没找到 App Server，试过：%s", strings.Join(plan.Tried, " → "))
}

// resolveExplicit 校验显式给出的可执行文件（绝对路径、相对路径或 PATH 里的名字）。
func resolveExplicit(plan serverPlan, source, value string) (serverPlan, error) {
	candidate := expandHome(value)
	if !strings.ContainsRune(candidate, filepath.Separator) {
		// 只给了名字：按 PATH 查。
		found, err := exec.LookPath(candidate)
		if err != nil {
			plan.Explicit = true
			return plan, fmt.Errorf("%s 给的 %q 在 PATH 里找不到", source, value)
		}
		candidate = found
	}
	ok, reason := executable(candidate)
	if !ok {
		if reason == "" {
			reason = "不存在"
		}
		plan.Explicit = true
		return plan, fmt.Errorf("%s 给的 App Server 不可用：%s（%s）", source, candidate, reason)
	}
	plan.Path, plan.Source = candidate, source
	return plan, nil
}

// executable 判断路径是不是可执行的普通文件；不存在时第二个返回值为空。
func executable(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		return false, ""
	}
	if info.IsDir() {
		return false, "是目录"
	}
	if info.Mode()&0o111 == 0 {
		return false, "不可执行"
	}
	return true, ""
}

// startupLog 是客户端自己的启动日志：追加写、只在启动与连接阶段用。
//
// 它不复制 App Server 的日志，也不读它的文件；内容只限启动参数、发现结果与
// 连接错误，供「查看日志」时定位。
type startupLog struct {
	mu   sync.Mutex
	file *os.File
	io.Writer
	// path 是日志文件路径；打开失败时为空。
	path string
}

// defaultLogPath 是默认日志路径：<数据目录>/logs/dimagent.log。
func defaultLogPath(dataDir string) string {
	return filepath.Join(dataDir, logDirName, logFileName)
}

// openStartupLog 打开日志；目录建不出来或文件打不开时退化为只写 stderr。
func openStartupLog(path string) *startupLog {
	log := &startupLog{path: path, Writer: os.Stderr}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "无法创建日志目录 %s: %v\n", filepath.Dir(path), err)
		return log
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法打开日志 %s: %v\n", path, err)
		return log
	}
	log.file = file
	log.Writer = io.MultiWriter(os.Stderr, file)
	return log
}

// Path 返回日志文件路径（打开失败时也返回预期路径，便于界面提示）。
func (l *startupLog) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// printf 写一行带时间戳的日志。行内不写凭据与内容正文。
func (l *startupLog) printf(format string, args ...any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.Writer, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// close 关闭日志文件；nil 安全。
func (l *startupLog) close() {
	if l == nil || l.file == nil {
		return
	}
	_ = l.file.Close()
	l.file = nil
}
