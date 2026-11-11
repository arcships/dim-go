package host

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// windowsInstaller 把替换交给 NSIS 安装器：退出五阶段后以 --updated /S 启动安装器，
// 由安装器完成文件替换与重启。
type windowsInstaller struct {
	hooks    InstallHooks
	forceRun bool
}

func (w *windowsInstaller) Prepare(_ context.Context, artifact string) (installPlan, error) {
	plan := installPlan{artifact: artifact}
	info, err := os.Stat(artifact)
	if err != nil || info.Size() == 0 {
		return plan, fmt.Errorf("host: 更新安装器不可用: %w", err)
	}
	if info.IsDir() {
		return plan, errors.New("host: 更新安装器必须是文件")
	}
	return plan, nil
}

func (w *windowsInstaller) Apply(ctx context.Context, plan *installPlan) error {
	return w.hooks.spawn(ctx, plan.artifact, nsisArgs(w.forceRun), os.Environ())
}

// Restart 由 NSIS 安装器负责，这里不再启动第二个进程。
func (w *windowsInstaller) Restart(context.Context, *installPlan) error { return nil }

func (w *windowsInstaller) Cleanup(*installPlan) {}

// nsisArgs 是 electron-updater 使用的一组 NSIS 参数：静默更新，可选完成后强制启动。
func nsisArgs(forceRun bool) []string {
	args := []string{"--updated", "/S"}
	if forceRun {
		args = append(args, "--force-run")
	}
	return args
}
