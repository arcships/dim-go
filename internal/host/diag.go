package host

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sync"
	"time"
)

// Level 是诊断记录的级别，只用于区分严重程度，不承载业务内容。
type Level string

const (
	// LevelInfo 是正常事件。
	LevelInfo Level = "info"
	// LevelWarn 是可恢复的异常。
	LevelWarn Level = "warn"
	// LevelError 是失败事件。
	LevelError Level = "error"
)

// Event 是诊断事件名。事件名走白名单形状校验，不含任何业务内容。
type Event string

const (
	// EventInstancePrimary 表示本进程占用了单实例端点。
	EventInstancePrimary Event = "instance.primary"
	// EventInstanceForwarded 表示二次启动已把参数转交给已有实例。
	EventInstanceForwarded Event = "instance.forwarded"
	// EventInstanceForwardFailed 表示二次启动转交失败。
	EventInstanceForwardFailed Event = "instance.forward_failed"
	// EventInstanceStaleRemoved 表示清理了被强杀后残留的实例端点。
	EventInstanceStaleRemoved Event = "instance.stale_removed"
	// EventActivationDropped 表示待处理接管队列已满，本次启动参数被丢弃。
	EventActivationDropped Event = "activation.dropped"
	// EventStartupStage 表示一个启动阶段完成。
	EventStartupStage Event = "startup.stage"
	// EventStartupFailed 表示启动停在某个阶段。
	EventStartupFailed Event = "startup.failed"
	// EventStartupReady 表示六阶段全部完成，可以接收写入。
	EventStartupReady Event = "startup.ready"
	// EventStageRejected 表示一次不合法的阶段推进已被丢弃。
	EventStageRejected Event = "startup.stage_rejected"
	// EventShutdownBegin 表示退出开始，不再接受新意图。
	EventShutdownBegin Event = "shutdown.begin"
	// EventShutdownPhase 表示一个退出阶段完成。
	EventShutdownPhase Event = "shutdown.phase"
	// EventShutdownTimeout 表示一个退出阶段超时并被跳过。
	EventShutdownTimeout Event = "shutdown.timeout"
	// EventShutdownDone 表示退出五阶段全部走完。
	EventShutdownDone Event = "shutdown.done"
	// EventConnectionChanged 表示观察到的连接状态变化（状态本身归连接领域）。
	EventConnectionChanged Event = "connection.changed"
)

// ErrorClass 是诊断里的错误分类，取值来自连接领域的五种类别加一个兜底。
type ErrorClass string

const (
	// ClassVersionIncompatible 表示协议版本不兼容。
	ClassVersionIncompatible ErrorClass = "version_incompatible"
	// ClassConnectionMissing 表示没有可用服务。
	ClassConnectionMissing ErrorClass = "connection_missing"
	// ClassTimeout 表示超时。
	ClassTimeout ErrorClass = "timeout"
	// ClassRetryable 表示可重试。
	ClassRetryable ErrorClass = "retryable"
	// ClassNonRetryable 表示不可重试。
	ClassNonRetryable ErrorClass = "non_retryable"
	// ClassOther 是兜底分类。
	ClassOther ErrorClass = "other"
)

// Outcome 是阶段或退出的结果。
type Outcome string

const (
	// OutcomeOK 表示完成。
	OutcomeOK Outcome = "ok"
	// OutcomeTimeout 表示超时。
	OutcomeTimeout Outcome = "timeout"
	// OutcomeFailed 表示失败。
	OutcomeFailed Outcome = "failed"
	// OutcomeSkipped 表示因不合法被丢弃。
	OutcomeSkipped Outcome = "skipped"
)

// redacted 是取值不合法时写进日志的固定占位符，绝不回填原值。
const redacted = "[redacted]"

// 白名单字段名。记录结构固定，未知字段没有落点。
const (
	fieldStage          = "stage"
	fieldFromStage      = "from_stage"
	fieldNextStage      = "next_stage"
	fieldPhase          = "phase"
	fieldOutcome        = "outcome"
	fieldClassification = "classification"
	fieldConnected      = "connected"
	fieldExiting        = "exiting"
	fieldVersion        = "version"
	fieldElapsed        = "elapsed_ms"
	fieldTimeout        = "timeout_ms"
	fieldArgs           = "args"
	fieldDropped        = "dropped"
)

// fieldKeys 是写入端允许出现的全部字段名；测试据此校验输出里没有越界键。
var fieldKeys = map[string]bool{
	fieldStage:          true,
	fieldFromStage:      true,
	fieldNextStage:      true,
	fieldPhase:          true,
	fieldOutcome:        true,
	fieldClassification: true,
	fieldConnected:      true,
	fieldExiting:        true,
	fieldVersion:        true,
	fieldElapsed:        true,
	fieldTimeout:        true,
	fieldArgs:           true,
	fieldDropped:        true,
}

// record 是落盘的一行诊断。字段集合就是白名单，没有自由文本字段。
type record struct {
	Time           string `json:"time"`
	Level          string `json:"level"`
	Event          string `json:"event"`
	Stage          string `json:"stage,omitempty"`
	FromStage      string `json:"from_stage,omitempty"`
	NextStage      string `json:"next_stage,omitempty"`
	Phase          string `json:"phase,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	Classification string `json:"classification,omitempty"`
	Connected      *bool  `json:"connected,omitempty"`
	Exiting        *bool  `json:"exiting,omitempty"`
	Version        string `json:"version,omitempty"`
	ElapsedMS      *int64 `json:"elapsed_ms,omitempty"`
	TimeoutMS      *int64 `json:"timeout_ms,omitempty"`
	Args           *int   `json:"args,omitempty"`
	Dropped        *int   `json:"dropped,omitempty"`
}

// valueKind 是字段取值的形状。形状决定校验方式，形状之外的字符串进不来。
type valueKind int

const (
	kindInvalid valueKind = iota
	kindStage
	kindPhase
	kindOutcome
	kindClass
	kindBool
	kindVersion
	kindMillis
	kindCount
)

// Field 是诊断记录里的一个字段。它只能用本包的构造函数生成：
// 字段名、取值形状与取值集合都受白名单约束，没有"任意文本"入口。
type Field struct {
	key  string
	kind valueKind
	text string
	num  int64
	flag bool
}

// StageField 写入当前阶段。
func StageField(stage Stage) Field {
	return Field{key: fieldStage, kind: kindStage, text: string(stage)}
}

// FromStageField 写入被拒绝迁移的起始阶段。
func FromStageField(stage Stage) Field {
	return Field{key: fieldFromStage, kind: kindStage, text: string(stage)}
}

// NextStageField 写入被拒绝迁移的目标阶段。
func NextStageField(stage Stage) Field {
	return Field{key: fieldNextStage, kind: kindStage, text: string(stage)}
}

// PhaseField 写入退出阶段。
func PhaseField(phase ShutdownPhase) Field {
	return Field{key: fieldPhase, kind: kindPhase, text: string(phase)}
}

// OutcomeField 写入结果。
func OutcomeField(outcome Outcome) Field {
	return Field{key: fieldOutcome, kind: kindOutcome, text: string(outcome)}
}

// ClassField 写入错误分类；未知分类一律记为 other，不保留原值。
func ClassField(class ErrorClass) Field {
	if !knownClasses[class] {
		return Field{key: fieldClassification, kind: kindClass, text: string(ClassOther)}
	}
	return Field{key: fieldClassification, kind: kindClass, text: string(class)}
}

// ConnectedField 写入"是否已连接"这一刻的观察值。
func ConnectedField(connected bool) Field {
	return Field{key: fieldConnected, kind: kindBool, flag: connected}
}

// ExitingField 写入"是否正在退出"。
func ExitingField(exiting bool) Field {
	return Field{key: fieldExiting, kind: kindBool, flag: exiting}
}

// VersionField 写入版本号。只接受 x.y.z 形式的版本串，其余一律遮蔽。
func VersionField(version string) Field {
	if !versionPattern.MatchString(version) {
		version = redacted
	}
	return Field{key: fieldVersion, kind: kindVersion, text: version}
}

// ElapsedField 写入耗时毫秒数；负值按 0 处理。
func ElapsedField(d time.Duration) Field {
	return Field{key: fieldElapsed, kind: kindMillis, num: nonNegativeMillis(d)}
}

// TimeoutField 写入超时毫秒数；负值按 0 处理。
func TimeoutField(d time.Duration) Field {
	return Field{key: fieldTimeout, kind: kindMillis, num: nonNegativeMillis(d)}
}

// ArgsField 写入参数个数；只记个数，不记参数内容。负值按 0 处理。
func ArgsField(n int) Field {
	if n < 0 {
		n = 0
	}
	return Field{key: fieldArgs, kind: kindCount, num: int64(n)}
}

// DroppedField 写入本次被丢弃的字段个数。
func DroppedField(n int) Field {
	if n < 0 {
		n = 0
	}
	return Field{key: fieldDropped, kind: kindCount, num: int64(n)}
}

func nonNegativeMillis(d time.Duration) int64 {
	ms := d.Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$`)
	eventPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

var (
	knownStages   = map[string]bool{}
	knownPhases   = map[string]bool{}
	knownOutcomes = map[string]bool{
		string(OutcomeOK):      true,
		string(OutcomeTimeout): true,
		string(OutcomeFailed):  true,
		string(OutcomeSkipped): true,
	}
	knownClasses = map[ErrorClass]bool{
		ClassVersionIncompatible: true,
		ClassConnectionMissing:   true,
		ClassTimeout:             true,
		ClassRetryable:           true,
		ClassNonRetryable:        true,
		ClassOther:               true,
	}
)

func init() {
	for _, stage := range append(StartupStages(), StageIdle, StageReady) {
		knownStages[string(stage)] = true
	}
	for _, phase := range ShutdownPhases() {
		knownPhases[string(phase)] = true
	}
}

// ErrorClassFrom 把外部错误分类名收敛到白名单，未知值返回 other。
func ErrorClassFrom(name string) ErrorClass {
	class := ErrorClass(name)
	if !knownClasses[class] {
		return ClassOther
	}
	return class
}

// Classified 是阶段错误的可选分类接口：错误自报分类名，host 只按白名单记录，
// 从不读取错误正文。连接领域可以在错误类型上实现它。
type Classified interface {
	ErrorClass() string
}

func classify(err error) ErrorClass {
	if err == nil {
		return ClassOther
	}
	var c Classified
	if errors.As(err, &c) {
		return ErrorClassFrom(c.ErrorClass())
	}
	return ClassOther
}

func (f Field) valid() bool {
	if f.key == "" || !fieldKeys[f.key] {
		return false
	}
	switch f.kind {
	case kindStage:
		return knownStages[f.text]
	case kindPhase:
		return knownPhases[f.text]
	case kindOutcome:
		return knownOutcomes[f.text]
	case kindClass:
		return knownClasses[ErrorClass(f.text)]
	case kindBool:
		return true
	case kindVersion:
		return f.text == redacted || versionPattern.MatchString(f.text)
	case kindMillis, kindCount:
		return f.num >= 0
	default:
		return false
	}
}

// apply 把一个字段写进记录；字段名与形状不匹配时返回 false，由调用方计入丢弃数。
func apply(rec *record, f Field) bool {
	if !f.valid() {
		return false
	}
	switch f.key {
	case fieldStage:
		rec.Stage = f.text
	case fieldFromStage:
		rec.FromStage = f.text
	case fieldNextStage:
		rec.NextStage = f.text
	case fieldPhase:
		rec.Phase = f.text
	case fieldOutcome:
		rec.Outcome = f.text
	case fieldClassification:
		rec.Classification = f.text
	case fieldConnected:
		v := f.flag
		rec.Connected = &v
	case fieldExiting:
		v := f.flag
		rec.Exiting = &v
	case fieldVersion:
		rec.Version = f.text
	case fieldElapsed:
		v := f.num
		rec.ElapsedMS = &v
	case fieldTimeout:
		v := f.num
		rec.TimeoutMS = &v
	case fieldArgs:
		v := int(f.num)
		rec.Args = &v
	case fieldDropped:
		v := int(f.num)
		rec.Dropped = &v
	default:
		return false
	}
	return true
}

// Recorder 是诊断记录的写入端：一条记录一行 JSON，字段受白名单约束，并发安全。
// 零值不可用，必须用 NewRecorder 构造；nil 接收者的方法全部安全空转。
type Recorder struct {
	mu           sync.Mutex
	w            io.Writer
	dropped      int
	writeFailure int
}

// NewRecorder 创建诊断写入端。w 为 nil 时丢弃全部记录。
func NewRecorder(w io.Writer) *Recorder {
	if w == nil {
		w = io.Discard
	}
	return &Recorder{w: w}
}

// Log 写一条诊断。event 形状不合法时记为 event.invalid；字段不合法时丢弃并计数，
// 记录里只出现 dropped 个数，不回填被丢弃的取值。
func (r *Recorder) Log(level Level, event Event, fields ...Field) {
	if r == nil {
		return
	}
	rec := record{
		Time:  time.Now().UTC().Format(time.RFC3339Nano),
		Level: string(level),
		Event: string(event),
	}
	switch level {
	case LevelInfo, LevelWarn, LevelError:
	default:
		rec.Level = string(LevelInfo)
	}
	if !eventPattern.MatchString(string(event)) || len(event) > 64 {
		rec.Event = "event.invalid"
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	rejected := 0
	for _, f := range fields {
		if apply(&rec, f) {
			continue
		}
		rejected++
	}
	if rejected > 0 {
		rec.Dropped = &rejected
		r.dropped += rejected
	}
	line, err := json.Marshal(rec)
	if err != nil {
		r.writeFailure++
		return
	}
	if _, err := r.w.Write(append(line, '\n')); err != nil {
		r.writeFailure++
	}
}

// Flush 冲刷底层写入端；底层支持 Flush 或 Sync 时一并调用。
func (r *Recorder) Flush() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	if f, ok := r.w.(interface{ Flush() error }); ok {
		errs = append(errs, f.Flush())
	}
	if f, ok := r.w.(interface{ Sync() error }); ok {
		errs = append(errs, f.Sync())
	}
	return errors.Join(errs...)
}

// Dropped 返回累计被丢弃的字段个数，供自检与测试使用。
func (r *Recorder) Dropped() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dropped
}

// WriteFailures 返回累计写入失败的记录条数。
func (r *Recorder) WriteFailures() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writeFailure
}
