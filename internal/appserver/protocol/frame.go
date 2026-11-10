package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// DefaultMaxFrameBytes 是单帧上限，与产品基线一致（16 MiB）。
const DefaultMaxFrameBytes = 16 * 1024 * 1024

var (
	// ErrFrameTooLarge 表示一帧超过上限。该帧被丢弃，解码器继续可用。
	ErrFrameTooLarge = errors.New("ndjson frame is too large")
	// ErrMalformedFrame 表示一帧不是合法 JSON。该帧被丢弃，解码器继续可用。
	ErrMalformedFrame = errors.New("malformed ndjson frame")
)

// FrameTooLargeError 带上超限帧的字节数与上限。
type FrameTooLargeError struct {
	Bytes    int
	MaxBytes int
}

func (e *FrameTooLargeError) Error() string {
	return fmt.Sprintf("ndjson frame is %d bytes, above the %d-byte limit", e.Bytes, e.MaxBytes)
}

func (e *FrameTooLargeError) Is(target error) bool { return target == ErrFrameTooLarge }

// MalformedFrameError 带底层 JSON 解析错误。
type MalformedFrameError struct {
	Err error
}

func (e *MalformedFrameError) Error() string {
	return fmt.Sprintf("malformed ndjson frame: %v", e.Err)
}

func (e *MalformedFrameError) Unwrap() error { return e.Err }

func (e *MalformedFrameError) Is(target error) bool { return target == ErrMalformedFrame }

// Decoder 按增量方式解析 NDJSON：任意分块边界都能接受，未结束的一行留在缓冲
// 里等换行。空行被忽略，行尾 `\r` 被去掉，UTF-8 之外的内容按 JSON 解析失败处理。
//
// 坏帧不会让解码器失去同步：超限或非法的整行被丢弃并报错一次，后续帧照常返回。
// 超限的未结束行进入丢弃状态，直到下一个换行，然后恢复正常。
type Decoder struct {
	maxBytes   int
	buf        []byte
	discarding bool
}

// NewDecoder 创建解码器；maxBytes 不大于 0 时使用 DefaultMaxFrameBytes。
func NewDecoder(maxBytes int) *Decoder {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	return &Decoder{maxBytes: maxBytes}
}

// Push 送入一段字节，返回其中解析出的完整帧。返回的帧是可安全保留的副本。
// 同一批里存在坏帧时，坏帧之前的帧仍会返回，错误只报第一条。
func (d *Decoder) Push(chunk []byte) ([][]byte, error) {
	var frames [][]byte
	var firstErr error
	record := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	for len(chunk) > 0 {
		idx := bytes.IndexByte(chunk, '\n')
		var line []byte
		if idx < 0 {
			line = chunk
			chunk = nil
		} else {
			line = chunk[:idx]
			chunk = chunk[idx+1:]
		}
		if d.discarding {
			if idx >= 0 {
				d.discarding = false
			}
			continue
		}
		d.buf = append(d.buf, line...)
		if idx < 0 {
			// 未结束的一行：只做上限检查。
			if len(d.buf) > d.maxBytes {
				record(&FrameTooLargeError{Bytes: len(d.buf), MaxBytes: d.maxBytes})
				d.buf = d.buf[:0]
				d.discarding = true
			}
			break
		}
		// 完整的一行。
		frame := bytes.TrimSuffix(d.buf, []byte("\r"))
		d.buf = d.buf[:0]
		if len(bytes.TrimSpace(frame)) == 0 {
			continue
		}
		if len(frame) > d.maxBytes {
			record(&FrameTooLargeError{Bytes: len(frame), MaxBytes: d.maxBytes})
			continue
		}
		if !json.Valid(frame) {
			record(&MalformedFrameError{Err: errors.New("invalid JSON")})
			continue
		}
		kept := make([]byte, len(frame))
		copy(kept, frame)
		frames = append(frames, kept)
	}
	return frames, firstErr
}

// Pending 返回尚未以换行结束的字节数。
func (d *Decoder) Pending() int { return len(d.buf) }

// EncodeFrame 把值编码为一帧（JSON 加换行）。
func EncodeFrame(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode ndjson frame: %w", err)
	}
	frame := make([]byte, 0, len(raw)+1)
	frame = append(frame, raw...)
	frame = append(frame, '\n')
	return frame, nil
}
