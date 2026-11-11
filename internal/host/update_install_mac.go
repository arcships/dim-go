package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// macInstaller 把更新 zip 解包成 .app，校验签名与公证后原子替换旧 .app，失败回滚。
type macInstaller struct {
	bundle string
	hooks  InstallHooks
	verify func(ctx context.Context, oldApp, newApp string) error
}

func (m *macInstaller) Prepare(ctx context.Context, artifact string) (installPlan, error) {
	plan := installPlan{artifact: artifact, target: m.bundle}

	info, err := os.Stat(m.bundle)
	if err != nil {
		return plan, fmt.Errorf("host: 当前 .app 不可用: %w", err)
	}
	if !info.IsDir() {
		return plan, errors.New("host: macOS 安装目标是 .app 目录")
	}
	artifactInfo, err := os.Stat(artifact)
	if err != nil {
		return plan, fmt.Errorf("host: 更新产物不可用: %w", err)
	}
	if artifactInfo.Size() == 0 {
		return plan, errors.New("host: 更新产物为空")
	}

	// 暂存目录必须和 .app 在同一卷，才能用 rename 原子替换。
	staging, err := os.MkdirTemp(filepath.Dir(m.bundle), ".dimagent-update-*")
	if err != nil {
		return plan, fmt.Errorf("host: 创建更新暂存目录失败: %w", err)
	}
	plan.staging = staging

	if err := m.unpack(ctx, artifact, staging); err != nil {
		m.Cleanup(&plan)
		return plan, err
	}
	newApp, err := findAppBundle(staging)
	if err != nil {
		m.Cleanup(&plan)
		return plan, err
	}
	plan.newApp = newApp

	verify := m.verify
	if verify == nil {
		verify = m.verifyWithCommands
	}
	if err := verify(ctx, m.bundle, newApp); err != nil {
		m.Cleanup(&plan)
		return plan, err
	}
	return plan, nil
}

func (m *macInstaller) unpack(ctx context.Context, artifact, staging string) error {
	// ditto 保留 .app 内的符号链接与扩展属性；缺失时退回 unzip。
	if _, err := m.hooks.run(ctx, "ditto", "-x", "-k", artifact, staging); err != nil {
		if !errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("host: 解包更新产物失败: %w", err)
		}
		if _, unzipErr := m.hooks.run(ctx, "unzip", "-q", artifact, "-d", staging); unzipErr != nil {
			return fmt.Errorf("host: 解包更新产物失败: %w", unzipErr)
		}
	}
	return nil
}

// findAppBundle 在解包目录里找唯一的 .app；zip 里的 __MACOSX 目录会被忽略。
func findAppBundle(staging string) (string, error) {
	entries, err := os.ReadDir(staging)
	if err != nil {
		return "", fmt.Errorf("host: 读取解包目录失败: %w", err)
	}
	var bundles []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "__MACOSX") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".app") {
			bundles = append(bundles, filepath.Join(staging, entry.Name()))
		}
	}
	if len(bundles) != 1 {
		return "", fmt.Errorf("host: 更新包应包含一个 .app，实际 %d 个", len(bundles))
	}
	return bundles[0], nil
}

// verifyWithCommands 是默认的签名与公证校验：
// codesign 校验签名完整性、TeamIdentifier 与当前 .app 一致、公证票据可校验、Gatekeeper 可通过。
func (m *macInstaller) verifyWithCommands(ctx context.Context, oldApp, newApp string) error {
	if _, err := m.hooks.run(ctx, "codesign", "--verify", "--deep", "--strict", "--verbose=2", newApp); err != nil {
		return fmt.Errorf("host: 更新包代码签名校验失败: %w", err)
	}
	oldTeam, err := m.teamIdentifier(ctx, oldApp)
	if err != nil {
		return err
	}
	newTeam, err := m.teamIdentifier(ctx, newApp)
	if err != nil {
		return err
	}
	if oldTeam == "" {
		return errors.New("host: 当前安装包没有 TeamIdentifier，无法校验更新包来源")
	}
	if newTeam == "" || !strings.EqualFold(oldTeam, newTeam) {
		return fmt.Errorf("host: 更新包 TeamIdentifier %q 与当前 %q 不一致", newTeam, oldTeam)
	}
	if _, err := m.hooks.run(ctx, "xcrun", "stapler", "validate", newApp); err != nil {
		return fmt.Errorf("host: 更新包公证票据校验失败: %w", err)
	}
	if _, err := m.hooks.run(ctx, "spctl", "--assess", "--type", "execute", "--verbose=4", newApp); err != nil {
		return fmt.Errorf("host: 更新包公证评估失败: %w", err)
	}
	return nil
}

func (m *macInstaller) teamIdentifier(ctx context.Context, app string) (string, error) {
	output, err := m.hooks.run(ctx, "codesign", "-d", "--verbose=4", app)
	if err != nil {
		return "", fmt.Errorf("host: 读取安装包签名信息失败: %w", err)
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TeamIdentifier=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "TeamIdentifier=")), nil
		}
	}
	return "", nil
}

func (m *macInstaller) Apply(_ context.Context, plan *installPlan) error {
	backup := plan.target + ".old-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.Rename(plan.target, backup); err != nil {
		return fmt.Errorf("host: 备份旧 .app 失败: %w", err)
	}
	plan.backup = backup
	if err := os.Rename(plan.newApp, plan.target); err != nil {
		// 新包挪不进去就把旧包换回来。
		if restoreErr := os.Rename(backup, plan.target); restoreErr != nil {
			return errors.Join(fmt.Errorf("host: 移入新 .app 失败: %w", err),
				fmt.Errorf("host: 恢复旧 .app 失败: %w", restoreErr))
		}
		plan.backup = ""
		return fmt.Errorf("host: 移入新 .app 失败: %w", err)
	}
	_ = os.RemoveAll(plan.staging)
	plan.staging = ""
	return nil
}

func (m *macInstaller) Restart(ctx context.Context, plan *installPlan) error {
	if err := m.hooks.spawn(ctx, "open", []string{"-n", plan.target}, os.Environ()); err != nil {
		rollbackErr := rollbackBundle(plan.target, plan.backup)
		return errors.Join(fmt.Errorf("host: 启动新版本失败: %w", err), rollbackErr)
	}
	_ = os.RemoveAll(plan.backup)
	plan.backup = ""
	return nil
}

// rollbackBundle 把已经放进目标位置的新 .app 挪走，再把备份换回来。
func rollbackBundle(target, backup string) error {
	if backup == "" {
		return nil
	}
	failed := target + ".failed-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.Rename(target, failed); err != nil {
		return fmt.Errorf("host: 回滚时移出新 .app 失败: %w", err)
	}
	if err := os.Rename(backup, target); err != nil {
		return fmt.Errorf("host: 回滚时恢复旧 .app 失败: %w", err)
	}
	_ = os.RemoveAll(failed)
	return nil
}

func (m *macInstaller) Cleanup(plan *installPlan) {
	if plan == nil || plan.staging == "" {
		return
	}
	_ = os.RemoveAll(plan.staging)
	plan.staging = ""
}
