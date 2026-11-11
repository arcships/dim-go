package app

import (
	"os"
	"path/filepath"
	"testing"
)

// 真实证据用例的路径来源：只从环境变量读取，缺变量就跳过并写明缺哪个。
//
// 不在代码里写本机路径或产品 home；证据环境由调用方设置环境变量提供。
const (
	envServerBinary  = "DIM_APP_SERVER_BIN"
	envRealFixture   = "DIM_REAL_MODEL_FIXTURE"
	envModelConfig   = "DIM_MODEL_CONFIG"
	envModalityImage = "DIM_MODALITY_IMAGE"
	envRuntimePreset = "DIM_RUNTIME_PRESET"
)

// evidenceRequiredEnv 读取必需的环境变量；缺失时跳过用例并点名变量。
func evidenceRequiredEnv(t *testing.T, name, purpose string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("缺少环境变量 %s（%s）；跳过真实证据", name, purpose)
	}
	return value
}

// evidenceServerBinary 返回真实 App Server 可执行文件路径。
func evidenceServerBinary(t *testing.T) string {
	t.Helper()
	return evidenceRequiredEnv(t, envServerBinary, "真实 App Server 可执行文件")
}

// evidenceFixtureHome 返回夹具的产品 home。
//
// DIM_REAL_MODEL_FIXTURE 指向夹具根（含 home/）或直接指向 home 都可以。
func evidenceFixtureHome(t *testing.T) string {
	t.Helper()
	root := evidenceRequiredEnv(t, envRealFixture, "真实模型夹具目录")
	home := filepath.Join(root, "home")
	if info, err := os.Stat(home); err == nil && info.IsDir() {
		return home
	}
	return root
}
