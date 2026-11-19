package host

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FeedUpdateState 是「关于 › 软件更新」的相位。它与设置页的界面分支一一对应：
//
//	idle       已配置更新源，尚未检查
//	checking   检查中
//	up_to_date 已是最新
//	available  有新版本可用（待下载）
//	downloading 下载中
//	downloaded 已下载并校验通过（待安装）
//	preparing  正在准备安装
//	error      检查或下载失败（带原因，可重试）
//
// 「未配置更新源」不是状态：没有配置时不构造 FeedUpdater，界面直接禁用。
type FeedUpdateState string

const (
	// FeedUpdateIdle 是已配置、未检查。
	FeedUpdateIdle FeedUpdateState = "idle"
	// FeedUpdateChecking 是检查中。
	FeedUpdateChecking FeedUpdateState = "checking"
	// FeedUpdateUpToDate 是已是最新。
	FeedUpdateUpToDate FeedUpdateState = "up_to_date"
	// FeedUpdateAvailable 是有新版本待下载。
	FeedUpdateAvailable FeedUpdateState = "available"
	// FeedUpdateDownloading 是下载中。
	FeedUpdateDownloading FeedUpdateState = "downloading"
	// FeedUpdateDownloaded 是已下载并校验通过。
	FeedUpdateDownloaded FeedUpdateState = "downloaded"
	// FeedUpdatePreparing 是安装准备中。
	FeedUpdatePreparing FeedUpdateState = "preparing"
	// FeedUpdateError 是检查、下载或安装失败。
	FeedUpdateError FeedUpdateState = "error"
)

var (
	// ErrFeedNotConfigured 表示没有配置更新源。
	ErrFeedNotConfigured = errors.New("host: 未配置更新源")
	// ErrFeedInstallUnavailable 表示当前宿主没有安装能力：入口保留但置灰。
	ErrFeedInstallUnavailable = errors.New("host: 当前版本不支持自动安装")
	// ErrFeedBusy 表示上一个检查或下载还没结束。
	ErrFeedBusy = errors.New("host: 更新操作尚未结束")
)

// FeedUpdateStatus 是更新相位的只读快照。
type FeedUpdateStatus struct {
	State            FeedUpdateState
	CurrentVersion   string
	AvailableVersion string
	Notes            string
	Percent          int
	ArtifactPath     string
	LastError        error
	Installable      bool
	InstallReason    string
}

// FeedUpdateConfig 是 JSON 版本源更新的构造参数。
type FeedUpdateConfig struct {
	// CurrentVersion 是当前构建版本，必须形如 x.y.z（可带预发布段）。
	CurrentVersion string
	// FeedURL 是设置里填写的更新源；基址会自动补 latest.json。
	FeedURL string
	// Dir 是下载落点（`<数据目录>/updates`）。
	Dir string
	// HTTPClient 可选；为空时用默认客户端。
	HTTPClient *http.Client
	// CheckTimeout / DownloadTimeout 为零时取默认值。
	CheckTimeout    time.Duration
	DownloadTimeout time.Duration
	// Install 是宿主安装能力（解包替换 .app 等）；为 nil 时安装入口置灰。
	Install func(ctx context.Context, artifactPath string) error
	// InstallReason 是安装不可用时的用户可见原因；为空时用缺省文案。
	InstallReason string
}

func (c FeedUpdateConfig) withDefaults() FeedUpdateConfig {
	if c.CheckTimeout <= 0 {
		c.CheckTimeout = 15 * time.Second
	}
	if c.DownloadTimeout <= 0 {
		c.DownloadTimeout = 10 * time.Minute
	}
	if strings.TrimSpace(c.InstallReason) == "" {
		c.InstallReason = "当前版本不支持自动安装"
	}
	return c
}

// FeedUpdater 实现 latest.json 版本源的检查、下载校验与安装分派。
//
// 它不持有界面状态，也不写诊断：失败原因放在 Status.LastError 里由界面呈现；
// 下载落在配置的目录（缺省由调用方给 `<数据目录>/updates`），校验不过不落盘。
type FeedUpdater struct {
	cfg        FeedUpdateConfig
	client     *http.Client
	feedURL    string
	dir        string
	installErr string

	opMu sync.Mutex
	mu   sync.Mutex

	state    FeedUpdateState
	feed     Feed
	percent  int
	artifact string
	partPath string
	lastErr  error
}

// NewFeedUpdater 校验参数并创建更新编排。未配置更新源时返回 ErrFeedNotConfigured，
// 由调用方按「未配置更新源」禁用入口。
func NewFeedUpdater(cfg FeedUpdateConfig) (*FeedUpdater, error) {
	cfg = cfg.withDefaults()
	current := strings.TrimSpace(cfg.CurrentVersion)
	if current == "" {
		return nil, errors.New("host: 更新缺少当前版本")
	}
	if _, err := parseSemver(current); err != nil {
		return nil, err
	}
	feedURL, err := NormalizeFeedURL(cfg.FeedURL)
	if err != nil {
		return nil, err
	}
	if feedURL == "" {
		return nil, ErrFeedNotConfigured
	}
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		return nil, errors.New("host: 更新缺少下载目录")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	installErr := strings.TrimSpace(cfg.InstallReason)
	return &FeedUpdater{
		cfg:        cfg,
		client:     client,
		feedURL:    feedURL,
		dir:        dir,
		installErr: installErr,
		state:      FeedUpdateIdle,
	}, nil
}

// FeedURL 返回规范化后的 latest.json 地址。
func (u *FeedUpdater) FeedURL() string {
	if u == nil {
		return ""
	}
	return u.feedURL
}

// State 返回当前相位。
func (u *FeedUpdater) State() FeedUpdateState {
	if u == nil {
		return FeedUpdateIdle
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state
}

// Status 返回只读快照。
func (u *FeedUpdater) Status() FeedUpdateStatus {
	if u == nil {
		return FeedUpdateStatus{}
	}
	u.mu.Lock()
	status := FeedUpdateStatus{
		State:            u.state,
		CurrentVersion:   strings.TrimSpace(u.cfg.CurrentVersion),
		AvailableVersion: u.feed.Version,
		Notes:            u.feed.Notes,
		Percent:          u.percent,
		ArtifactPath:     u.artifact,
		LastError:        u.lastErr,
		Installable:      u.cfg.Install != nil,
		InstallReason:    u.installErr,
	}
	u.mu.Unlock()
	return status
}

// Check 读取 latest.json 并比较版本。可重复调用：失败后再调用就是「重试」。
// 低于 minVersion 时不提供增量更新，直接报原因。
func (u *FeedUpdater) Check(ctx context.Context) (FeedUpdateStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	if err := u.begin(FeedUpdateChecking); err != nil {
		return u.Status(), err
	}
	u.mu.Lock()
	// 新一轮检查先丢掉上一次的发布事实：检查失败时不把旧版本当成「待下载」。
	u.feed = Feed{}
	u.artifact = ""
	u.percent = 0
	u.lastErr = nil
	u.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, u.cfg.CheckTimeout)
	defer cancel()
	feed, err := u.fetchFeed(ctx)
	if err != nil {
		u.fail(err)
		return u.Status(), err
	}
	current := strings.TrimSpace(u.cfg.CurrentVersion)
	if feed.MinVersion != "" {
		compared, err := compareVersions(current, feed.MinVersion)
		if err != nil {
			u.fail(err)
			return u.Status(), err
		}
		if compared < 0 {
			err := fmt.Errorf("当前版本 %s 低于更新源要求的最低版本 %s，需要重新安装", current, feed.MinVersion)
			u.fail(err)
			return u.Status(), err
		}
	}
	compared, err := compareVersions(current, feed.Version)
	if err != nil {
		u.fail(err)
		return u.Status(), err
	}

	u.mu.Lock()
	u.lastErr = nil
	u.percent = 0
	u.artifact = ""
	if compared < 0 {
		u.feed = feed
		u.state = FeedUpdateAvailable
	} else {
		// 已是最新：不保留可下载的发布事实，界面不显示「新版本」行。
		u.feed = Feed{Notes: feed.Notes}
		u.state = FeedUpdateUpToDate
	}
	u.mu.Unlock()
	return u.Status(), nil
}

// Download 下载产物到 Dir，并按 latest.json 的 sha256 校验。失败时保留「有新版本」
// 的事实，原因进 LastError，可直接重试下载。
func (u *FeedUpdater) Download(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	if err := u.begin(FeedUpdateDownloading); err != nil {
		return "", err
	}
	u.mu.Lock()
	feed := u.feed
	u.percent = 0
	u.lastErr = nil
	u.artifact = ""
	u.mu.Unlock()
	if feed.Version == "" || feed.URL == "" {
		err := errors.New("host: 先检查更新再下载")
		u.fail(err)
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, u.cfg.DownloadTimeout)
	defer cancel()
	artifact, err := u.download(ctx, feed)
	if err != nil {
		u.fail(err)
		return "", err
	}
	u.mu.Lock()
	u.state = FeedUpdateDownloaded
	u.artifact = artifact
	u.percent = 100
	u.partPath = ""
	u.mu.Unlock()
	return artifact, nil
}

// Install 调用宿主安装能力。没有安装能力时返回 ErrFeedInstallUnavailable 并带上
// 用户可见原因；不写假回执。
func (u *FeedUpdater) Install(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if u == nil || u.cfg.Install == nil {
		return fmt.Errorf("%w: %s", ErrFeedInstallUnavailable, u.installReason())
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	u.mu.Lock()
	if u.state != FeedUpdateDownloaded || u.artifact == "" {
		u.mu.Unlock()
		return fmt.Errorf("%w: 安装（当前 %s）", ErrUpdateState, u.state)
	}
	artifact := u.artifact
	u.state = FeedUpdatePreparing
	u.lastErr = nil
	u.mu.Unlock()

	if err := u.cfg.Install(ctx, artifact); err != nil {
		u.fail(err)
		return err
	}
	u.mu.Lock()
	// 安装完成后进程应由宿主重启；这里保留「已下载」相位与产物路径，不假装重启过。
	u.state = FeedUpdateDownloaded
	u.mu.Unlock()
	return nil
}

// Close 清理未完成的下载临时文件；已校验的产物保留在 Dir 里等安装。
func (u *FeedUpdater) Close() error {
	if u == nil {
		return nil
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	u.mu.Lock()
	part := u.partPath
	u.partPath = ""
	u.mu.Unlock()
	if part == "" {
		return nil
	}
	return removeIfExists(part)
}

func (u *FeedUpdater) installReason() string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.installErr) != "" {
		return u.installErr
	}
	return "当前版本不支持自动安装"
}

// begin 把相位推进到操作态；已有操作在跑时拒绝并保留现状。
func (u *FeedUpdater) begin(next FeedUpdateState) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	switch u.state {
	case FeedUpdateChecking, FeedUpdateDownloading, FeedUpdatePreparing:
		return fmt.Errorf("%w: 当前 %s", ErrFeedBusy, u.state)
	}
	u.state = next
	return nil
}

func (u *FeedUpdater) fail(err error) {
	u.mu.Lock()
	u.state = FeedUpdateError
	u.lastErr = err
	u.percent = 0
	u.mu.Unlock()
}

func (u *FeedUpdater) fetchFeed(ctx context.Context) (Feed, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.feedURL, nil)
	if err != nil {
		return Feed{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := u.client.Do(request)
	if err != nil {
		return Feed{}, classifyFeedError(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Feed{}, fmt.Errorf("更新源返回 %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxFeedBytes+1))
	if err != nil {
		return Feed{}, classifyFeedError(err)
	}
	if int64(len(body)) > maxFeedBytes {
		return Feed{}, errors.New("更新源超过读取上限")
	}
	return ParseFeed(u.feedURL, body)
}

func (u *FeedUpdater) download(ctx context.Context, feed Feed) (string, error) {
	expected, err := decodeSHA256(feed.SHA256)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(u.dir, 0o700); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		return "", err
	}
	response, err := u.client.Do(request)
	if err != nil {
		return "", classifyFeedError(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载新版本返回 %d", response.StatusCode)
	}
	if response.ContentLength > maxArtifactBytes {
		return "", fmt.Errorf("产物大小 %d 超过上限", response.ContentLength)
	}

	name := sanitizeFileName(feed.Version + "-" + artifactFileName(feed.URL))
	target := filepath.Join(u.dir, name)
	part := target + ".part"
	u.mu.Lock()
	u.partPath = part
	u.mu.Unlock()

	file, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	progress := &feedProgress{
		writer: io.MultiWriter(file, hasher),
		total:  response.ContentLength,
		report: func(percent int) {
			u.mu.Lock()
			if u.state == FeedUpdateDownloading {
				u.percent = percent
			}
			u.mu.Unlock()
		},
	}
	written, copyErr := io.Copy(progress, io.LimitReader(response.Body, maxArtifactBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		u.dropPart(part)
		return "", classifyFeedError(copyErr)
	}
	if closeErr != nil {
		u.dropPart(part)
		return "", closeErr
	}
	if written > maxArtifactBytes {
		u.dropPart(part)
		return "", errors.New("产物超过下载上限")
	}
	if response.ContentLength > 0 && written != response.ContentLength {
		u.dropPart(part)
		return "", fmt.Errorf("产物大小 %d，更新源写的是 %d", written, response.ContentLength)
	}
	if subtle.ConstantTimeCompare(hasher.Sum(nil), expected) != 1 {
		u.dropPart(part)
		return "", errors.New("下载校验失败：sha256 摘要与更新源不一致")
	}
	if err := os.Rename(part, target); err != nil {
		u.dropPart(part)
		return "", err
	}
	return target, nil
}

func (u *FeedUpdater) dropPart(part string) {
	_ = os.Remove(part)
	u.mu.Lock()
	if u.partPath == part {
		u.partPath = ""
	}
	u.mu.Unlock()
}

// classifyFeedError 只把超时与取消原样保留，其余统一成可读的网络失败原因。
func classifyFeedError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("更新源请求超时")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("更新源请求已取消")
	}
	return fmt.Errorf("连接更新源失败: %w", err)
}

// feedProgress 在写下载文件的同时按内容长度报进度。
type feedProgress struct {
	writer  io.Writer
	total   int64
	written int64
	report  func(int)
}

func (p *feedProgress) Write(data []byte) (int, error) {
	written, err := p.writer.Write(data)
	p.written += int64(written)
	if p.total > 0 && p.report != nil {
		percent := int(p.written * 100 / p.total)
		if percent > 100 {
			percent = 100
		}
		p.report(percent)
	}
	return written, err
}
