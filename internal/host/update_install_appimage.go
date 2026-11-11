package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// linuxInstaller 原地替换 AppImage：同目录写临时文件、chmod 0755、rename 覆盖，
// 成功后带 APPIMAGE_SILENT_INSTALL=1 启动新文件。
type linuxInstaller struct {
	target string
	hooks  InstallHooks
}

func (l *linuxInstaller) Prepare(_ context.Context, artifact string) (installPlan, error) {
	plan := installPlan{artifact: artifact, target: l.target}
	// 目标是符号链接时替换真实文件，不能把链接本身换掉。
	resolved, err := filepath.EvalSymlinks(l.target)
	if err != nil {
		return plan, fmt.Errorf("host: 当前 AppImage 不可用: %w", err)
	}
	plan.target = resolved
	info, err := os.Stat(resolved)
	if err != nil {
		return plan, fmt.Errorf("host: 当前 AppImage 不可用: %w", err)
	}
	if !info.Mode().IsRegular() {
		return plan, errors.New("host: Linux 安装目标必须是 AppImage 文件")
	}
	artifactInfo, err := os.Stat(artifact)
	if err != nil {
		return plan, fmt.Errorf("host: 更新产物不可用: %w", err)
	}
	if artifactInfo.Size() == 0 {
		return plan, errors.New("host: 更新产物为空")
	}
	return plan, nil
}

func (l *linuxInstaller) Apply(_ context.Context, plan *installPlan) error {
	return installExecutable(plan.artifact, plan.target, 0o755)
}

func (l *linuxInstaller) Restart(ctx context.Context, plan *installPlan) error {
	env := withEnv(os.Environ(), "APPIMAGE_SILENT_INSTALL", "1")
	return l.hooks.spawn(ctx, plan.target, nil, env)
}

func (l *linuxInstaller) Cleanup(*installPlan) {}
