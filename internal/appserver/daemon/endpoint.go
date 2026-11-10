// Package daemon 负责 App Server 的本地端点、自启与回收、握手校验。
//
// 它只做生命周期与连接，不解释业务方法；会话流见 session 包。
package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

// Config 是连接配置。DataDir 是客户端自己的数据目录，App Server 的运行时
// 文件只放在它下面的 rust 子目录；本客户端不读其中的数据库与会话文件。
type Config struct {
	DataDir     string
	ServerPath  string
	ServerArgs  []string
	ClientName  string
	MockModel   bool
	ModelConfig string

	ConnectTimeout time.Duration
	SpawnTimeout   time.Duration
	RequestTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.ClientName == "" {
		c.ClientName = "dimagent-mygo"
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	if c.SpawnTimeout <= 0 {
		c.SpawnTimeout = 10 * time.Second
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = 30 * time.Second
	}
	return c
}

// endpoint 是本机端点与服务端 home 的解析结果。
type endpoint struct {
	// nativeHome 是 App Server 的 `--home`，即数据目录下的 rust 子目录。
	nativeHome string
	network    transport.Endpoint
}

// resolveEndpoint 按数据目录解析端点并确保目录私有。
func resolveEndpoint(dataDir string) (endpoint, error) {
	if dataDir == "" {
		return endpoint{}, errors.New("连接配置缺少数据目录")
	}
	nativeHome := filepath.Join(dataDir, "rust")
	if err := ensurePrivateDir(nativeHome); err != nil {
		return endpoint{}, fmt.Errorf("准备 App Server home: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(nativeHome)
	if err != nil {
		return endpoint{}, fmt.Errorf("解析 App Server home: %w", err)
	}
	if runtime.GOOS == "windows" {
		return endpoint{
			nativeHome: resolved,
			network: transport.Endpoint{
				Network: transport.NetworkPipe,
				Address: windowsPipeName(resolved),
			},
		}, nil
	}
	return endpoint{
		nativeHome: resolved,
		network: transport.Endpoint{
			Network: transport.NetworkUnix,
			Address: filepath.Join(resolved, "daemon.sock"),
		},
	}, nil
}

// windowsPipeName 与产品基线使用同一个命名规则，避免两个客户端各连一个管道。
func windowsPipeName(nativeHome string) string {
	sum := sha256.Sum256([]byte("dimcode:" + nativeHome))
	return `\\.\pipe\dim-rust-` + hex.EncodeToString(sum[:])[:32]
}

func ensurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return checkPrivateDir(path)
}

// buildDaemonArgs 组装自启参数。与产品基线一致：显式给出端点、关掉 Web 入口，
// 有模型配置时用配置，否则用产品 home；测试与证据可用 mock 模型。
func buildDaemonArgs(cfg Config, ep endpoint) []string {
	args := append([]string{}, cfg.ServerArgs...)
	if ep.network.Network == transport.NetworkPipe {
		args = append(args, "--pipe", ep.network.Address)
	} else {
		args = append(args, "--unix", ep.network.Address)
	}
	args = append(args, "--no-ws", "--home", ep.nativeHome)
	if cfg.MockModel {
		args = append(args, "--mock-model")
	}
	if cfg.ModelConfig != "" {
		args = append(args, "--model-config", cfg.ModelConfig)
	} else if !cfg.MockModel {
		args = append(args, "--product-home", cfg.DataDir)
	}
	return args
}
