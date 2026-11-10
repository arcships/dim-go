// Package attachments 负责附件的本地暂存。
//
// 范围内的职责：把选定的本地文件复制进暂存区，返回暂存标识与元数据；按内容摘要去重；
// 在暂存时完成单文件与暂存区总大小上限判定；按内容区分文本与图片；识别失效暂存项并清理。
// 不包含：提交与上传（属写登记）、界面、会话读取与工作区读取。
//
// 暂存区位于应用数据目录下的独立子目录，只保存文件与索引元数据，不保存会话、草稿等业务事实。
// 暂存区内的文件权限为 0600，目录权限为 0700。
package attachments

import (
	"errors"
	"fmt"
	"time"
)

const (
	// DirName 是暂存区在应用数据目录下的子目录名。
	DirName = "attachment-staging"
	// indexFileName 是暂存区索引文件名。
	indexFileName = "index.json"
	// blobDirName 是内容文件子目录名。
	blobDirName = "blobs"
	// indexVersion 是索引格式版本。格式或类别语义发生不兼容变化时递增。
	indexVersion = 1
	// sniffLen 是判定类别时读取的文件头字节数。
	sniffLen = 512

	// 需求基线 R7 拍板的口径是文本 10 MB、图片 8 MB，超限在提交前拒绝；
	// 这里按十进制 MB 折算，数值可由调用方通过 Limits 覆盖。
	defaultTextLimit  int64 = 10 * 1000 * 1000
	defaultImageLimit int64 = 8 * 1000 * 1000
	// 暂存区总大小上限需求未规定，默认 500 MB，仅用于防止暂存区无界增长。
	defaultTotalLimit int64 = 500 * 1000 * 1000
)

// Category 是附件类别，决定上层用资源引用还是内联内容。
type Category string

const (
	// CategoryText 是文本附件，上层按服务端资源引用提交。
	CategoryText Category = "text"
	// CategoryImage 是图片附件，上层按图片内容提交。
	CategoryImage Category = "image"
)

func (c Category) valid() bool {
	return c == CategoryText || c == CategoryImage
}

func (c Category) label() string {
	if c == CategoryImage {
		return "图片"
	}
	return "文本"
}

// Limits 是暂存上限。所有字段都必须大于 0。
type Limits struct {
	// Text 是单个文本附件的字节上限。
	Text int64
	// Image 是单个图片附件的字节上限。
	Image int64
	// Total 是暂存区去重后内容总量的字节上限。
	Total int64
}

// DefaultLimits 返回需求基线与本领域约定的默认上限。
func DefaultLimits() Limits {
	return Limits{
		Text:  defaultTextLimit,
		Image: defaultImageLimit,
		Total: defaultTotalLimit,
	}
}

func (l Limits) validate() error {
	switch {
	case l.Text <= 0:
		return errors.New("文本附件单文件上限必须大于 0")
	case l.Image <= 0:
		return errors.New("图片附件单文件上限必须大于 0")
	case l.Total <= 0:
		return errors.New("暂存区总大小上限必须大于 0")
	}
	return nil
}

func (l Limits) perFile(c Category) int64 {
	if c == CategoryImage {
		return l.Image
	}
	return l.Text
}

// Entry 是一条暂存记录的元数据，也是暂存项的对外视图。
type Entry struct {
	// ID 是暂存标识，上层用它引用附件并去重上传。
	ID string `json:"id"`
	// Digest 是内容的 SHA-256 摘要，十六进制小写，也是去重键。
	Digest string `json:"digest"`
	// Name 是暂存时的文件名（不含目录）。
	Name string `json:"name"`
	// Category 是按内容判定的类别。
	Category Category `json:"category"`
	// Size 是内容字节数。
	Size int64 `json:"size"`
	// Refs 是挂载引用计数：每次成功暂存加一，每次释放减一。
	Refs int `json:"refs"`
	// StagedAt 是该内容首次进入暂存区的时间（UTC）。
	StagedAt time.Time `json:"staged_at"`
}

// Problem 是暂存项失效的原因；空值表示该项有效。
type Problem string

const (
	// ProblemNone 表示暂存项有效。
	ProblemNone Problem = ""
	// ProblemRecordInvalid 表示索引记录本身不合法。
	ProblemRecordInvalid Problem = "record_invalid"
	// ProblemBlobMissing 表示内容文件缺失。
	ProblemBlobMissing Problem = "blob_missing"
	// ProblemBlobNotRegular 表示内容路径不是普通文件。
	ProblemBlobNotRegular Problem = "blob_not_regular"
	// ProblemBlobUnreadable 表示内容文件读不出来。
	ProblemBlobUnreadable Problem = "blob_unreadable"
	// ProblemSizeMismatch 表示内容文件字节数与记录不一致。
	ProblemSizeMismatch Problem = "size_mismatch"
	// ProblemDigestMismatch 表示内容摘要与记录不一致。
	ProblemDigestMismatch Problem = "digest_mismatch"
	// ProblemCategoryMismatch 表示按内容判定的类别与记录不一致。
	ProblemCategoryMismatch Problem = "category_mismatch"
)

// Check 是一次暂存项校验结果。
type Check struct {
	Entry   Entry   `json:"entry"`
	Problem Problem `json:"problem"`
	Detail  string  `json:"detail,omitempty"`
}

// OK 表示该项有效。
func (c Check) OK() bool { return c.Problem == ProblemNone }

// CleanupReport 是一次清理的结果。
type CleanupReport struct {
	// RemovedInvalid 是因失效且无引用被删除的暂存项标识。
	RemovedInvalid []string
	// RemovedUnreferenced 是因无引用被删除的有效暂存项标识。
	RemovedUnreferenced []string
	// KeptReferenced 是失效但仍被引用、因此未删除的暂存项。
	KeptReferenced []Check
	// RemovedOrphans 是被删除的无索引记录内容文件（不含目录名）。
	RemovedOrphans []string
	// BytesFreed 是本次释放的字节数。
	BytesFreed int64
}

// LimitKind 区分单文件上限与暂存区总大小上限。
type LimitKind string

const (
	// LimitKindFile 是单文件上限。
	LimitKindFile LimitKind = "file"
	// LimitKindTotal 是暂存区总大小上限。
	LimitKindTotal LimitKind = "total"
)

// LimitError 表示附件超过上限。判定在暂存时完成，调用方可直接取 Size 与 Limit 展示。
type LimitError struct {
	Kind     LimitKind
	Category Category
	// Size 是本次文件的字节数。
	Size int64
	// Current 是判定时暂存区已占用的字节数，仅总大小上限使用。
	Current int64
	// Limit 是被触发的上限字节数。
	Limit int64
}

func (e *LimitError) Error() string {
	if e.Kind == LimitKindTotal {
		return fmt.Sprintf("附件超过暂存区总大小上限：已占用 %d 字节，本次%s附件 %d 字节，合计 %d 字节，上限 %d 字节",
			e.Current, e.Category.label(), e.Size, e.Current+e.Size, e.Limit)
	}
	return fmt.Sprintf("附件超过单文件上限：%s附件 %d 字节，上限 %d 字节",
		e.Category.label(), e.Size, e.Limit)
}

// UnsupportedError 表示附件内容既不是文本也不是图片，因此无法暂存。
type UnsupportedError struct {
	Name   string
	Size   int64
	Reason string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("附件内容不受支持：%s（%d 字节）：%s", e.Name, e.Size, e.Reason)
}

// Is 让 errors.Is(err, ErrUnsupported) 成立。
func (e *UnsupportedError) Is(target error) bool { return target == ErrUnsupported }

var (
	// ErrNotFound 表示按标识找不到暂存项。
	ErrNotFound = errors.New("暂存项不存在")
	// ErrNotRegular 表示给定路径不是普通文件。
	ErrNotRegular = errors.New("附件路径不是普通文件")
	// ErrUnsupported 表示附件内容既不是文本也不是图片。
	ErrUnsupported = errors.New("附件内容不受支持")
	// ErrIndexCorrupt 表示暂存索引无法解析或结构不合法。
	ErrIndexCorrupt = errors.New("暂存索引损坏")
	// ErrIndexVersion 表示暂存索引版本不在当前支持范围内。
	ErrIndexVersion = errors.New("暂存索引版本不受支持")
)
