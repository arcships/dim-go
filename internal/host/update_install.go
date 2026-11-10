package host

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
	if err := os.Chmod(tempName, info.Mode().Perm()); err != nil {
		return fmt.Errorf("host: 设置更新产物权限失败: %w", err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("host: 替换安装产物失败: %w", err)
	}
	renamed = true
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
