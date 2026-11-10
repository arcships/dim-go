package host

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// maxArtifactBytes 是单个更新产物的下载上限，防止版本源或网络错误写满磁盘。
	maxArtifactBytes int64 = 1 << 30
)

var (
	// ErrUpdateState 表示当前更新状态不允许该操作，事件已被丢弃。
	ErrUpdateState = errors.New("host: 更新状态不允许该操作")
	// ErrSessionsBusy 表示有会话尚未收尾，本次切换被推迟。
	ErrSessionsBusy = errors.New("host: 有会话未收尾，推迟切换")
)

// UpdateState 是更新状态。取值与 `A.md` 的「更新」迁移表一一对应。
type UpdateState string

const (
	// UpdateUnchecked 是未检查。
	UpdateUnchecked UpdateState = "unchecked"
	// UpdateChecking 是检查中。
	UpdateChecking UpdateState = "checking"
	// UpdateAvailable 是待下载（有新版）。
	UpdateAvailable UpdateState = "available"
	// UpdateUpToDate 是已是最新；检查失败也按此呈现。
	UpdateUpToDate UpdateState = "up_to_date"
	// UpdateReady 是待切换（下载并校验完成）。
	UpdateReady UpdateState = "ready_to_switch"
	// UpdateSwitching 是切换中（终态，直到进程重启）。
	UpdateSwitching UpdateState = "switching"
)

// UpdateStates 返回全部更新状态。
func UpdateStates() []UpdateState {
	return []UpdateState{
		UpdateUnchecked,
		UpdateChecking,
		UpdateAvailable,
		UpdateUpToDate,
		UpdateReady,
		UpdateSwitching,
	}
}

// updateTransitions 与 `A.md` 的「更新」迁移表逐条对应：
//
//	未检查 + 检查版本 → 检查中
//	检查中 + 有新版 → 待下载
//	检查中 + 已是最新或检查失败 → 已是最新
//	待下载 + 下载并校验完成 → 待切换
//	待切换 + 用户确认且会话已收尾 → 切换中
var updateTransitions = map[UpdateState]map[UpdateState]bool{
	UpdateUnchecked: {UpdateChecking: true},
	UpdateChecking:  {UpdateAvailable: true, UpdateUpToDate: true},
	UpdateAvailable: {UpdateReady: true},
	UpdateUpToDate:  {},
	UpdateReady:     {UpdateSwitching: true},
	UpdateSwitching: {},
}

func canUpdateTransition(from, to UpdateState) bool {
	allowed, ok := updateTransitions[from]
	return ok && allowed[to]
}

// UpdateStatus 是更新状态的只读快照。
type UpdateStatus struct {
	// State 是当前更新状态。
	State UpdateState
	// CurrentVersion 是当前版本；配置了 ReadVersion 时每次读取。
	CurrentVersion string
	// AvailableVersion 是版本源给出的新版本；无新版时为空。
	AvailableVersion string
	// ArtifactPath 是已下载并校验过的产物路径；未下载时为空。
	ArtifactPath string
	// CheckedAt 是最近一次检查结束的时间。
	CheckedAt time.Time
	// LastError 是最近一次检查、下载或切换的失败原因，仅供界面呈现，不写诊断。
	LastError error
}

// UpdateTimeouts 是更新各步骤的超时；小于等于 0 时取默认值。
type UpdateTimeouts struct {
	Check    time.Duration
	Download time.Duration
	Settle   time.Duration
	Switch   time.Duration
}

// DefaultUpdateTimeouts 是默认的更新超时。
func DefaultUpdateTimeouts() UpdateTimeouts {
	return UpdateTimeouts{
		Check:    15 * time.Second,
		Download: 10 * time.Minute,
		Settle:   30 * time.Second,
		Switch:   5 * time.Minute,
	}
}

func (t UpdateTimeouts) withDefaults() UpdateTimeouts {
	defaults := DefaultUpdateTimeouts()
	if t.Check <= 0 {
		t.Check = defaults.Check
	}
	if t.Download <= 0 {
		t.Download = defaults.Download
	}
	if t.Settle <= 0 {
		t.Settle = defaults.Settle
	}
	if t.Switch <= 0 {
		t.Switch = defaults.Switch
	}
	return t
}

// UpdateConfig 是更新编排的构造参数。
type UpdateConfig struct {
	// CurrentVersion 是当前版本，必须是 x.y.z 形式。
	CurrentVersion string
	// ReadVersion 可选：每次读取当前版本（例如从已安装产物读取）。设置后优先于 CurrentVersion。
	ReadVersion func() (string, error)
	// ManifestURL 是版本清单完整地址；为空时按 BaseURL/ChannelPath/Platform/Arch 推导。
	ManifestURL string
	// BaseURL 是更新源基址，默认取现有客户端的更新源。
	BaseURL string
	// ChannelPath 是更新通道路径，默认 updates/stable。
	ChannelPath string
	// Platform 默认取 runtime.GOOS。
	Platform string
	// Arch 默认取 runtime.GOARCH。
	Arch string
	// HTTPClient 可选；为空时使用默认客户端。
	HTTPClient *http.Client
	// TempDir 是下载落点；为空时新建临时目录并由 Close 回收。
	TempDir string
	// TargetPath 是要替换的安装产物路径（本版只支持单个文件）。
	TargetPath string
	// SessionsIdle 是会话是否已收尾的只读探针；为 nil 时视为已收尾。
	SessionsIdle func() bool
	// Settle 在检测到会话未收尾时先行收尾；为 nil 时直接推迟切换。
	Settle func(context.Context) error
	// Shutdown 可选：切换前的退出五阶段编排；为 nil 时跳过。
	Shutdown func(context.Context) error
	// Restart 是替换后启动新进程的步骤；切换时必须接线。
	Restart func(context.Context) error
	// Diag 是诊断写入端，可为 nil。
	Diag *Recorder
	// Timeouts 是各步骤超时。
	Timeouts UpdateTimeouts
}

// Updater 实现 `A.md` 的更新状态机：检查、下载校验、待切换到切换。
// 连接、界面与发布渠道都不在这里。
type Updater struct {
	cfg         UpdateConfig
	diag        *Recorder
	client      *http.Client
	manifestURL string
	tempDir     string
	ownsTemp    bool
	timeouts    UpdateTimeouts

	opMu sync.Mutex
	mu   sync.Mutex

	state        UpdateState
	release      Release
	artifactPath string
	checkedAt    time.Time
	lastErr      error
}

// NewUpdater 校验并创建更新编排。不支持更新的平台在这里返回错误，由调用方跳过更新。
func NewUpdater(cfg UpdateConfig) (*Updater, error) {
	if strings.TrimSpace(cfg.CurrentVersion) == "" && cfg.ReadVersion == nil {
		return nil, errors.New("host: 更新缺少当前版本")
	}
	if cfg.CurrentVersion != "" {
		if _, err := parseSemver(cfg.CurrentVersion); err != nil {
			return nil, err
		}
	}
	platform := cfg.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	arch := cfg.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	manifestURL := strings.TrimSpace(cfg.ManifestURL)
	if manifestURL == "" {
		derived, err := UpdateManifestURL(cfg.BaseURL, cfg.ChannelPath, platform, arch)
		if err != nil {
			return nil, err
		}
		manifestURL = derived
	}
	parsed, err := url.Parse(manifestURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("host: 版本清单地址必须是 http 或 https: %q", manifestURL)
	}

	tempDir := strings.TrimSpace(cfg.TempDir)
	ownsTemp := false
	if tempDir == "" {
		created, err := os.MkdirTemp("", "dimagent-update-")
		if err != nil {
			return nil, fmt.Errorf("host: 创建更新临时目录失败: %w", err)
		}
		tempDir, ownsTemp = created, true
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &Updater{
		cfg:         cfg,
		diag:        cfg.Diag,
		client:      client,
		manifestURL: manifestURL,
		tempDir:     tempDir,
		ownsTemp:    ownsTemp,
		timeouts:    cfg.Timeouts.withDefaults(),
		state:       UpdateUnchecked,
	}, nil
}

// State 返回当前更新状态。
func (u *Updater) State() UpdateState {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state
}

// Status 返回更新状态的只读快照。
func (u *Updater) Status() UpdateStatus {
	u.mu.Lock()
	status := UpdateStatus{
		State:            u.state,
		AvailableVersion: u.release.Version,
		ArtifactPath:     u.artifactPath,
		CheckedAt:        u.checkedAt,
		LastError:        u.lastErr,
	}
	u.mu.Unlock()
	status.CurrentVersion = u.currentVersion()
	return status
}

// Check 检查版本源。有新版进入待下载；已是最新或检查失败都进入已是最新，
// 失败原因记录在诊断与 Status.LastError 里，不阻塞使用。
func (u *Updater) Check(ctx context.Context) (UpdateStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	if !u.advance(UpdateUnchecked, UpdateChecking) {
		return u.Status(), fmt.Errorf("%w: 检查（当前 %s）", ErrUpdateState, u.State())
	}

	ctx, cancel := context.WithTimeout(ctx, u.timeouts.Check)
	defer cancel()
	start := time.Now()

	release, err := u.fetchRelease(ctx)
	if err == nil {
		var compared int
		compared, err = compareVersions(u.currentVersion(), release.Version)
		if err == nil && compared < 0 {
			if !u.advance(UpdateChecking, UpdateAvailable) {
				return u.Status(), fmt.Errorf("%w: 检查（当前 %s）", ErrUpdateState, u.State())
			}
			u.finishCheck(release, nil)
			u.diag.Log(LevelInfo, EventUpdateCheck,
				OutcomeField(OutcomeOK), UpdateStateField(UpdateAvailable),
				UpdateVersionField(release.Version), ElapsedField(time.Since(start)))
			return u.Status(), nil
		}
	}
	if err != nil {
		if !u.advance(UpdateChecking, UpdateUpToDate) {
			return u.Status(), fmt.Errorf("%w: 检查（当前 %s）", ErrUpdateState, u.State())
		}
		u.finishCheck(Release{}, err)
		u.diag.Log(LevelWarn, EventUpdateCheck,
			OutcomeField(OutcomeFailed), UpdateStateField(UpdateUpToDate),
			ClassField(classify(err)), ElapsedField(time.Since(start)))
		return u.Status(), err
	}

	if !u.advance(UpdateChecking, UpdateUpToDate) {
		return u.Status(), fmt.Errorf("%w: 检查（当前 %s）", ErrUpdateState, u.State())
	}
	u.finishCheck(Release{}, nil)
	u.diag.Log(LevelInfo, EventUpdateCheck,
		OutcomeField(OutcomeOK), UpdateStateField(UpdateUpToDate), ElapsedField(time.Since(start)))
	return u.Status(), nil
}

// Download 下载并在临时目录里校验产物。失败时保留"待下载"状态与产物地址信息，
// 当前版本不变；成功后进入待切换。
func (u *Updater) Download(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	if !u.begin(UpdateAvailable) {
		return "", fmt.Errorf("%w: 下载（当前 %s）", ErrUpdateState, u.State())
	}
	release := u.currentRelease()

	ctx, cancel := context.WithTimeout(ctx, u.timeouts.Download)
	defer cancel()
	start := time.Now()

	artifact, err := u.downloadRelease(ctx, release)
	if err != nil {
		u.setLastError(err)
		u.diag.Log(LevelError, EventUpdateDownload,
			OutcomeField(OutcomeFailed), UpdateVersionField(release.Version),
			ClassField(classify(err)), ElapsedField(time.Since(start)))
		return "", err
	}

	if !u.advance(UpdateAvailable, UpdateReady) {
		return "", fmt.Errorf("%w: 下载（当前 %s）", ErrUpdateState, u.State())
	}
	u.mu.Lock()
	u.artifactPath = artifact
	u.lastErr = nil
	u.mu.Unlock()
	u.diag.Log(LevelInfo, EventUpdateDownload,
		OutcomeField(OutcomeOK), UpdateVersionField(release.Version), ElapsedField(time.Since(start)))
	return artifact, nil
}

// Switch 执行"用户确认且会话已收尾"后的切换：先收尾会话，再退出五阶段（若接线），
// 替换安装产物，最后启动新进程。会话始终收不干净时推迟，状态保持待切换。
func (u *Updater) Switch(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	if !u.begin(UpdateReady) {
		return fmt.Errorf("%w: 切换（当前 %s）", ErrUpdateState, u.State())
	}
	release := u.currentRelease()

	if strings.TrimSpace(u.cfg.TargetPath) == "" || u.cfg.Restart == nil {
		err := updateFailure(ClassNonRetryable, errors.New("host: 切换缺少安装路径或重启步骤"))
		u.setLastError(err)
		u.diag.Log(LevelError, EventUpdateSwitch,
			OutcomeField(OutcomeFailed), UpdateVersionField(release.Version), ClassField(ClassNonRetryable))
		return err
	}

	if !u.sessionsIdle() {
		settleErr := u.settleSessions(ctx)
		if !u.sessionsIdle() {
			fields := []Field{
				UpdateStateField(UpdateReady),
				UpdateVersionField(release.Version),
				OutcomeField(OutcomeSkipped),
			}
			if settleErr != nil {
				fields = append(fields, ClassField(classify(settleErr)))
			}
			u.diag.Log(LevelWarn, EventUpdateSwitchDeferred, fields...)
			return ErrSessionsBusy
		}
	}

	if !u.advance(UpdateReady, UpdateSwitching) {
		return fmt.Errorf("%w: 切换（当前 %s）", ErrUpdateState, u.State())
	}
	u.mu.Lock()
	artifact := u.artifactPath
	u.mu.Unlock()

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, u.timeouts.Switch)
	defer cancel()

	if u.cfg.Shutdown != nil {
		if err := u.cfg.Shutdown(ctx); err != nil {
			u.failSwitch(release.Version, start, updateFailure(ClassOther, err))
			return err
		}
	}
	if err := installArtifact(artifact, u.cfg.TargetPath); err != nil {
		u.failSwitch(release.Version, start, updateFailure(ClassNonRetryable, err))
		return err
	}
	if err := u.cfg.Restart(ctx); err != nil {
		u.failSwitch(release.Version, start, updateFailure(ClassOther, err))
		return err
	}

	u.diag.Log(LevelInfo, EventUpdateSwitch,
		OutcomeField(OutcomeOK), UpdateVersionField(release.Version), ElapsedField(time.Since(start)))
	_ = u.diag.Flush()
	return nil
}

// Close 清理已下载的产物；临时目录由本包创建时一并删除。
func (u *Updater) Close() error {
	if u == nil {
		return nil
	}
	u.opMu.Lock()
	defer u.opMu.Unlock()
	var errs []error
	if u.artifactPath != "" {
		if err := removeIfExists(u.artifactPath); err != nil {
			errs = append(errs, err)
		}
	}
	if u.ownsTemp {
		if err := os.RemoveAll(u.tempDir); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// begin 校验操作的前置状态；不满足时丢弃事件并写诊断。
func (u *Updater) begin(want UpdateState) bool {
	current := u.State()
	if current == want {
		return true
	}
	u.diag.Log(LevelWarn, EventUpdateTransitionRejected,
		UpdateStateField(current), OutcomeField(OutcomeSkipped))
	return false
}

// advance 按 `A.md` 的迁移表推进状态；不合法时丢弃并写诊断。
func (u *Updater) advance(from, to UpdateState) bool {
	u.mu.Lock()
	if u.state == from && canUpdateTransition(from, to) {
		u.state = to
		u.mu.Unlock()
		return true
	}
	current := u.state
	u.mu.Unlock()
	u.diag.Log(LevelWarn, EventUpdateTransitionRejected,
		UpdateStateField(current), OutcomeField(OutcomeSkipped))
	return false
}

// finishCheck 记录一次检查的收尾字段；状态已由 advance 推进。
func (u *Updater) finishCheck(release Release, err error) {
	u.mu.Lock()
	u.release = release
	u.artifactPath = ""
	u.checkedAt = time.Now()
	u.lastErr = err
	u.mu.Unlock()
}

func (u *Updater) currentRelease() Release {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.release
}

func (u *Updater) setLastError(err error) {
	u.mu.Lock()
	u.lastErr = err
	u.mu.Unlock()
}

func (u *Updater) failSwitch(version string, start time.Time, err error) {
	u.setLastError(err)
	u.diag.Log(LevelError, EventUpdateSwitch,
		OutcomeField(OutcomeFailed), UpdateVersionField(version),
		ClassField(classify(err)), ElapsedField(time.Since(start)))
	_ = u.diag.Flush()
}

func (u *Updater) currentVersion() string {
	if u.cfg.ReadVersion != nil {
		if version, err := u.cfg.ReadVersion(); err == nil {
			if trimmed := strings.TrimSpace(version); trimmed != "" {
				return trimmed
			}
		}
	}
	return strings.TrimSpace(u.cfg.CurrentVersion)
}

func (u *Updater) sessionsIdle() bool {
	if u.cfg.SessionsIdle == nil {
		return true
	}
	return u.cfg.SessionsIdle()
}

func (u *Updater) settleSessions(parent context.Context) error {
	if u.cfg.Settle == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), u.timeouts.Settle)
	defer cancel()
	return u.cfg.Settle(ctx)
}

func (u *Updater) fetchRelease(ctx context.Context) (Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.manifestURL, nil)
	if err != nil {
		return Release{}, updateFailure(ClassNonRetryable, err)
	}
	request.Header.Set("Accept", "text/yaml,text/plain;q=0.9,*/*;q=0.1")
	response, err := u.client.Do(request)
	if err != nil {
		return Release{}, updateFailure(classifyRequestError(err), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Release{}, updateFailure(classifyStatus(response.StatusCode),
			fmt.Errorf("host: 版本源返回 %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxManifestBytes+1))
	if err != nil {
		return Release{}, updateFailure(classifyRequestError(err), err)
	}
	if int64(len(body)) > maxManifestBytes {
		return Release{}, updateFailure(ClassNonRetryable, errors.New("host: 版本清单超过上限"))
	}
	release, err := ParseManifest(u.manifestURL, body)
	if err != nil {
		return Release{}, updateFailure(ClassNonRetryable, err)
	}
	return release, nil
}

func (u *Updater) downloadRelease(ctx context.Context, release Release) (string, error) {
	if release.ArtifactURL == "" {
		return "", updateFailure(ClassNonRetryable, errors.New("host: 版本清单缺少产物地址"))
	}
	expected, err := decodeDigest(release.SHA512)
	if err != nil {
		return "", updateFailure(ClassNonRetryable, err)
	}
	if release.Size > maxArtifactBytes {
		return "", updateFailure(ClassNonRetryable,
			fmt.Errorf("host: 产物大小 %d 超过上限 %d", release.Size, maxArtifactBytes))
	}
	if err := os.MkdirAll(u.tempDir, 0o700); err != nil {
		return "", updateFailure(ClassOther, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.ArtifactURL, nil)
	if err != nil {
		return "", updateFailure(ClassNonRetryable, err)
	}
	response, err := u.client.Do(request)
	if err != nil {
		return "", updateFailure(classifyRequestError(err), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", updateFailure(classifyStatus(response.StatusCode),
			fmt.Errorf("host: 产物下载返回 %d", response.StatusCode))
	}

	limit := maxArtifactBytes + 1
	if release.Size > 0 {
		limit = release.Size + 1
	}
	target := filepath.Join(u.tempDir, artifactFileName(release.ArtifactURL))
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", updateFailure(ClassOther, err)
	}
	hasher := sha512.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(response.Body, limit))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(target)
		return "", updateFailure(classifyRequestError(copyErr), copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return "", updateFailure(ClassOther, closeErr)
	}
	if release.Size > 0 && written != release.Size {
		_ = os.Remove(target)
		return "", updateFailure(ClassNonRetryable,
			fmt.Errorf("host: 产物大小 %d，版本清单写的是 %d", written, release.Size))
	}
	if written > maxArtifactBytes {
		_ = os.Remove(target)
		return "", updateFailure(ClassNonRetryable, errors.New("host: 产物超过下载上限"))
	}
	if subtle.ConstantTimeCompare(hasher.Sum(nil), expected) != 1 {
		_ = os.Remove(target)
		return "", updateFailure(ClassNonRetryable, errors.New("host: 产物摘要与版本清单不一致"))
	}
	return target, nil
}

func classifyRequestError(err error) ErrorClass {
	switch {
	case err == nil:
		return ClassOther
	case errors.Is(err, context.DeadlineExceeded):
		return ClassTimeout
	case errors.Is(err, context.Canceled):
		return ClassRetryable
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ClassTimeout
	}
	return ClassConnectionMissing
}

func classifyStatus(status int) ErrorClass {
	switch {
	case status == http.StatusNotFound:
		return ClassNonRetryable
	case status == http.StatusTooManyRequests:
		return ClassRetryable
	case status >= 500:
		return ClassRetryable
	case status >= 400:
		return ClassNonRetryable
	default:
		return ClassOther
	}
}

func artifactFileName(rawURL string) string {
	if parsed, err := url.Parse(rawURL); err == nil {
		if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
			return sanitizeFileName(base)
		}
	}
	return "update-artifact"
}

func sanitizeFileName(name string) string {
	var builder strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	if builder.Len() == 0 {
		return "update-artifact"
	}
	return builder.String()
}

// updateError 让更新失败带上白名单里的错误分类，诊断只记分类不记正文。
type updateError struct {
	class ErrorClass
	err   error
}

func updateFailure(class ErrorClass, err error) error {
	if err == nil {
		return nil
	}
	return &updateError{class: ErrorClassFrom(string(class)), err: err}
}

func (e *updateError) Error() string { return e.err.Error() }

func (e *updateError) Unwrap() error { return e.err }

func (e *updateError) ErrorClass() string { return string(e.class) }
