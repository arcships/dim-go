package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/host"
)

// updateUsage 是 update 子命令的用法。
const updateUsage = `用法：dimagent update check|download|switch [参数]

子命令：
  check     检查版本源；检查失败不阻塞使用（退出码 0，只打印原因）
  download  检查并下载校验产物；失败退出码 1
  switch    检查、下载并切换（有会话未收尾时推迟并明确报错）

参数：
  -data-dir <目录>  客户端数据目录；缺省用 DIMAGENT_DATA_DIR 或系统默认
  -version <x.y.z>  当前版本；缺省从构建信息读取，读不到用 0.0.0
  -base <地址>      更新源基址；缺省用现有客户端的默认渠道
  -channel <路径>   更新通道；缺省 updates/stable
  -manifest <文件或地址> 完整清单地址；本地文件按 file:// 读取
  -target <文件>    要替换的单文件产物；缺省是当前可执行文件
  -temp <目录>      下载落点；缺省由更新编排自建并回收
`

// runUpdate 执行 update 子命令；只打印状态与版本号，不打印下载地址。
func runUpdate(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, updateUsage)
		return exitUsage
	}
	action := args[0]
	switch action {
	case "check", "download", "switch":
	case "-h", "--help", "help":
		fmt.Print(updateUsage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "dimagent update: 未知动作 %q\n", action)
		fmt.Fprint(os.Stderr, updateUsage)
		return exitUsage
	}

	flags := flag.NewFlagSet("update "+action, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	dataDir := flags.String("data-dir", defaultDataDir(), "客户端数据目录")
	version := flags.String("version", defaultBuildVersion(), "当前版本")
	base := flags.String("base", "", "更新源基址")
	channel := flags.String("channel", "", "更新通道")
	manifest := flags.String("manifest", "", "版本清单地址或本地文件")
	target := flags.String("target", "", "要替换的单文件产物")
	temp := flags.String("temp", "", "下载落点")
	if err := flags.Parse(args[1:]); err != nil {
		return exitUsage
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dimagent update: 无法定位当前可执行文件: %v\n", err)
		return exitFailed
	}
	targetPath := *target
	if targetPath == "" {
		targetPath = executable
	}
	if *temp == "" {
		*temp = filepath.Join(*dataDir, "update-temp")
	}

	baseDomain := *base
	process, err := app.New(app.Config{
		DataDir:    *dataDir,
		ClientName: "dimagent-update",
		Version:    *version,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dimagent update: 创建集成层: %v\n", err)
		return exitFailed
	}
	defer func() { _ = process.CloseUpdate() }()
	manifestURL := *manifest
	if manifestURL != "" && !looksLikeURL(manifestURL) {
		manifestURL = "file://" + filepath.ToSlash(manifestURL)
	}
	updater, err := process.EnableUpdate(app.UpdateOptions{
		CurrentVersion: *version,
		BaseURL:        baseDomain,
		ChannelPath:    *channel,
		ManifestURL:    manifestURL,
		TempDir:        *temp,
		TargetPath:     targetPath,
		Restart: func(ctx context.Context) error {
			// 切换后启动新进程：只启动，不接管输出。
			command := exec.CommandContext(ctx, targetPath)
			command.Args = []string{targetPath}
			return command.Start()
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dimagent update: 不支持更新或参数不合法: %v\n", err)
		return exitFailed
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if action == "check" {
		status, checkErr := updater.Check(ctx)
		printUpdateStatus(status)
		if checkErr != nil {
			fmt.Fprintf(os.Stderr, "dimagent update: 检查失败（不阻塞使用）: %v\n", checkErr)
		}
		return 0
	}

	if status, checkErr := updater.Check(ctx); checkErr != nil {
		printUpdateStatus(status)
		fmt.Fprintf(os.Stderr, "dimagent update: 检查失败，无法继续: %v\n", checkErr)
		return exitFailed
	}
	if action == "download" || statusOf(updater) == host.UpdateAvailable {
		if _, err := updater.Download(ctx); err != nil {
			printUpdateStatus(updater.Status())
			fmt.Fprintf(os.Stderr, "dimagent update: 下载或校验失败: %v\n", err)
			return exitFailed
		}
		artifact := updater.Status().ArtifactPath
		if info, statErr := os.Stat(artifact); statErr == nil {
			fmt.Printf("artifact: bytes=%d\n", info.Size())
		}
	}
	if action == "download" {
		printUpdateStatus(updater.Status())
		return 0
	}

	if err := updater.Switch(ctx); err != nil {
		printUpdateStatus(updater.Status())
		if errors.Is(err, host.ErrSessionsBusy) {
			fmt.Fprintln(os.Stderr, "dimagent update: 有会话未收尾，切换已推迟；收尾后重试")
			return exitFailed
		}
		fmt.Fprintf(os.Stderr, "dimagent update: 切换失败: %v\n", err)
		return exitFailed
	}
	printUpdateStatus(updater.Status())
	return 0
}

// printUpdateStatus 只打印状态与版本号；不打印清单或产物地址。
func printUpdateStatus(status host.UpdateStatus) {
	fmt.Printf("update: state=%s current=%s available=%s\n",
		status.State, emptyIfMissing(status.CurrentVersion), emptyIfMissing(status.AvailableVersion))
}

func emptyIfMissing(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func statusOf(updater *host.Updater) host.UpdateState {
	if updater == nil {
		return host.UpdateUnchecked
	}
	return updater.Status().State
}

func looksLikeURL(raw string) bool {
	return len(raw) > 7 && (raw[:7] == "http://" || (len(raw) > 8 && raw[:8] == "https://")) || len(raw) > 5 && raw[:5] == "file:"
}

// defaultBuildVersion 从构建信息读取主模块版本；读不到用 0.0.0 占位。
func defaultBuildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "0.0.0"
	}
	return info.Main.Version
}

// defaultDataDir 取 DIMAGENT_DATA_DIR，缺省放到用户数据目录下。
func defaultDataDir() string {
	if value := os.Getenv("DIMAGENT_DATA_DIR"); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dimagent"
	}
	return filepath.Join(home, ".dimagent")
}
