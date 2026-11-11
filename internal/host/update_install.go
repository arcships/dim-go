package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallMode 决定 Switch 使用哪个安装适配器。
type InstallMode string

const (
	// InstallAuto 按当前平台分派：darwin → .app，linux → AppImage，windows → NSIS，其余 → 单文件。
	InstallAuto InstallMode = "auto"
	// InstallFile 是单文件原子替换，供测试或非标准安装使用；需要 TargetPath 与 Restart。
	InstallFile InstallMode = "file"
	// InstallMacBundle 是 macOS zip → .app 的安装适配器。
	InstallMacBundle InstallMode = "mac_bundle"
	// InstallLinuxAppImage 是 Linux AppImage 的原地替换适配器。
	InstallLinuxAppImage InstallMode = "linux_appimage"
	// InstallWindowsNSIS 是 Windows NSIS 安装器的执行适配器。
	InstallWindowsNSIS InstallMode = "windows_nsis"
)

func installModeForPlatform(platform string) InstallMode {
	switch platform {
	case "darwin":
		return InstallMacBundle
	case "linux":
		return InstallLinuxAppImage
	case "windows":
		return InstallWindowsNSIS
	default:
		return InstallFile
	}
}

// InstallHooks 是安装适配器使用的外部命令与进程钩子。
// Run 执行命令并等待结束；Spawn 启动子进程后立即返回。
// 为零时分别使用系统命令与脱离父进程的默认实现；测试注入它们以跳过真实系统调用。
type InstallHooks struct {
	Run   func(ctx context.Context, name string, args ...string) (string, error)
	Spawn func(ctx context.Context, name string, args []string, env []string) error
}

func (h InstallHooks) run(ctx context.Context, name string, args ...string) (string, error) {
	if h.Run != nil {
		return h.Run(ctx, name, args...)
	}
	return runCommand(ctx, name, args...)
}

func (h InstallHooks) spawn(ctx context.Context, name string, args []string, env []string) error {
	if h.Spawn != nil {
		return h.Spawn(ctx, name, args, env)
	}
	return spawnProcess(name, args, env)
}

// installPlan 是 Prepare 与 Apply/Restart 之间传递的安装计划。
type installPlan struct {
	artifact string
	target   string
	newApp   string
	staging  string
	backup   string
}

// installer 是平台安装适配器。Prepare 只读校验，Apply 在退出后改安装位置，Restart 启动新版本。
type installer interface {
	Prepare(ctx context.Context, artifact string) (installPlan, error)
	Apply(ctx context.Context, plan *installPlan) error
	Restart(ctx context.Context, plan *installPlan) error
	Cleanup(plan *installPlan)
}

// newInstaller 按配置或平台选择安装适配器。
func (u *Updater) newInstaller() (installer, error) {
	mode := u.cfg.InstallMode
	if mode == "" || mode == InstallAuto {
		mode = installModeForPlatform(u.platform)
	}
	switch mode {
	case InstallFile:
		if strings.TrimSpace(u.cfg.TargetPath) == "" || u.cfg.Restart == nil {
			return nil, errors.New("host: 单文件安装缺少目标路径或重启步骤")
		}
		return &fileInstaller{target: u.cfg.TargetPath, restart: u.cfg.Restart}, nil
	case InstallMacBundle:
		if strings.TrimSpace(u.cfg.TargetPath) == "" {
			return nil, errors.New("host: macOS 安装缺少 .app 路径")
		}
		return &macInstaller{bundle: u.cfg.TargetPath, hooks: u.cfg.InstallHooks, verify: u.cfg.VerifySignature}, nil
	case InstallLinuxAppImage:
		if strings.TrimSpace(u.cfg.TargetPath) == "" {
			return nil, errors.New("host: Linux 安装缺少 AppImage 路径")
		}
		return &linuxInstaller{target: u.cfg.TargetPath, hooks: u.cfg.InstallHooks}, nil
	case InstallWindowsNSIS:
		return &windowsInstaller{hooks: u.cfg.InstallHooks, forceRun: u.cfg.ForceRunAfterInstall}, nil
	default:
		return nil, fmt.Errorf("host: 未知安装模式 %q", mode)
	}
}

// fileInstaller 保留单文件原子替换语义，供测试与非标准安装使用。
type fileInstaller struct {
	target  string
	restart func(context.Context) error
}

func (f *fileInstaller) Prepare(_ context.Context, artifact string) (installPlan, error) {
	info, err := os.Stat(f.target)
	if err != nil {
		return installPlan{}, fmt.Errorf("host: 安装路径不可用: %w", err)
	}
	if info.IsDir() {
		return installPlan{}, errors.New("host: 单文件安装的目标不能是目录")
	}
	if info, err := os.Stat(artifact); err != nil || info.Size() == 0 {
		return installPlan{}, fmt.Errorf("host: 更新产物不可用: %w", err)
	}
	return installPlan{artifact: artifact, target: f.target}, nil
}

func (f *fileInstaller) Apply(_ context.Context, plan *installPlan) error {
	return installArtifact(plan.artifact, plan.target)
}

func (f *fileInstaller) Restart(ctx context.Context, _ *installPlan) error {
	return f.restart(ctx)
}

func (f *fileInstaller) Cleanup(*installPlan) {}

// installArtifact 把已下载并校验过的单文件产物原子替换到目标路径：
// 先写同目录临时文件、同步、改权限，再 rename 覆盖，避免留下半截文件。
func installArtifact(artifact, target string) error {
	if strings.TrimSpace(artifact) == "" {
		return errors.New("host: 没有已校验的更新产物")
	}
	if strings.TrimSpace(target) == "" {
		return errors.New("host: 没有安装路径")
	}
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("host: 安装路径不可用: %w", err)
	}
	if info.IsDir() {
		return errors.New("host: 安装路径是目录，本版只替换单文件产物")
	}
	return installExecutable(artifact, target, info.Mode().Perm())
}

// installExecutable 用同目录临时文件 + rename 原子替换一个可执行文件，并设置指定权限。
func installExecutable(artifact, target string, mode os.FileMode) error {
	source, err := os.Open(artifact)
	if err != nil {
		return fmt.Errorf("host: 打不开更新产物: %w", err)
	}
	defer source.Close()

	temp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".new-*")
	if err != nil {
		return fmt.Errorf("host: 创建替换临时文件失败: %w", err)
	}
	tempName := temp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tempName)
		}
	}()

	if _, err := io.Copy(temp, source); err != nil {
		_ = temp.Close()
		return fmt.Errorf("host: 写入更新产物失败: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("host: 同步更新产物失败: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("host: 关闭更新产物失败: %w", err)
	}
	if err := os.Chmod(tempName, mode); err != nil {
		return fmt.Errorf("host: 设置更新产物权限失败: %w", err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("host: 替换安装产物失败: %w", err)
	}
	renamed = true
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("host: 命令 %s 失败: %w", name, err)
	}
	return string(output), nil
}

func spawnProcess(name string, args []string, env []string) error {
	command := exec.Command(name, args...)
	if env != nil {
		command.Env = env
	}
	configureDetached(command)
	if err := command.Start(); err != nil {
		return fmt.Errorf("host: 启动 %s 失败: %w", name, err)
	}
	return nil
}

// removeIfExists 删除文件；不存在不算错误。
func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
