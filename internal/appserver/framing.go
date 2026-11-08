// Package appserver contains the protocol-v3 transport primitives.
package appserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MaxFrameSize is the maximum number of JSON bytes in one NDJSON frame.
const MaxFrameSize = 16 * 1024 * 1024

var ErrFrameTooLarge = errors.New("appserver: frame exceeds 16 MiB")

// FrameReader reads LF-terminated NDJSON frames. It accepts CRLF and ignores
// blank lines, while never allowing an unterminated residual frame to grow
// beyond MaxFrameSize.
type FrameReader struct {
	r *bufio.Reader
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{r: bufio.NewReader(r)}
}

// ReadFrame returns the next non-empty JSON line. A stream ending in a partial
// line is reported as io.ErrUnexpectedEOF because protocol frames require LF.
func (r *FrameReader) ReadFrame() ([]byte, error) {
	var frame []byte
	for {
		part, err := r.r.ReadSlice('\n')
		if len(part) > 0 {
			frame = append(frame, part...)
			line := part
			if len(frame) > MaxFrameSize+1 { // allow one LF while checking below
				return nil, ErrFrameTooLarge
			}
			if err == nil {
				line = frame
				if len(line) > 0 && line[len(line)-1] == '\n' {
					line = line[:len(line)-1]
				}
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				if len(line) > MaxFrameSize {
					return nil, ErrFrameTooLarge
				}
				frame = frame[:0]
				if len(bytes.TrimSpace(line)) == 0 {
					continue
				}
				return append([]byte(nil), line...), nil
			}
		}
		if err != nil {
			if errors.Is(err, bufio.ErrBufferFull) {
				if len(frame) > MaxFrameSize {
					return nil, ErrFrameTooLarge
				}
				continue
			}
			if errors.Is(err, io.EOF) {
				if len(frame) == 0 {
					return nil, io.EOF
				}
				if len(frame) > MaxFrameSize {
					return nil, ErrFrameTooLarge
				}
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
	}
}

// ValidateEnvelope checks the protocol-v3 JSON-RPC envelope without imposing
// schemas on method parameters, results, or notification payloads.
func ValidateEnvelope(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var fields map[string]json.RawMessage
	if err := dec.Decode(&fields); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values in frame")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	var version string
	if err := json.Unmarshal(fields["jsonrpc"], &version); err != nil || version != "2.0" {
		return errors.New("jsonrpc must be \"2.0\"")
	}
	_, hasMethod := fields["method"]
	_, hasID := fields["id"]
	_, hasResult := fields["result"]
	_, hasError := fields["error"]
	if hasMethod {
		var method string
		if err := json.Unmarshal(fields["method"], &method); err != nil || method == "" {
			return errors.New("method must be a non-empty string")
		}
		if hasResult || hasError {
			return errors.New("request/notification must not contain result or error")
		}
		if hasID {
			if string(fields["id"]) == "null" {
				return errors.New("request id must be a number or string")
			}
			return validateID(fields["id"])
		}
		return nil
	}
	if !hasID || hasResult == hasError {
		return errors.New("reply must contain exactly one of result or error and an id")
	}
	if string(fields["id"]) == "null" {
		if !hasError {
			return errors.New("null id is only valid on error replies")
		}
	} else if err := validateID(fields["id"]); err != nil {
		return err
	}
	if hasError {
		var e struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(fields["error"], &e); err != nil {
			return fmt.Errorf("invalid error object: %w", err)
		}
		if len(e.Data) > 0 && string(e.Data) != "null" {
			var data struct {
				AppError string `json:"appError"`
			}
			if err := json.Unmarshal(e.Data, &data); err != nil || data.AppError == "" {
				return errors.New("error.data.appError must be a string")
			}
		}
	}
	return nil
}

func validateID(raw json.RawMessage) error {
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return nil
	}
	return errors.New("id must be a number or string")
}
