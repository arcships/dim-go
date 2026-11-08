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

// maxLineSize is MaxFrameSize plus the optional CRLF terminator, which is not
// part of the JSON payload.
const maxLineSize = MaxFrameSize + 2

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
			if len(frame) > maxLineSize {
				return nil, ErrFrameTooLarge
			}
			if err == nil {
				line := frame[:len(frame)-1] // drop LF
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
				// One trailing CR may still be waiting for its LF; anything
				// longer can no longer become a valid frame.
				if len(frame) > MaxFrameSize+1 {
					return nil, ErrFrameTooLarge
				}
				continue
			}
			if errors.Is(err, io.EOF) {
				if len(frame) == 0 {
					return nil, io.EOF
				}
				line := frame
				if line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				if len(line) > MaxFrameSize {
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
			if isJSONNull(fields["id"]) {
				return errors.New("request id must be a number or string")
			}
			return validateID(fields["id"])
		}
		return nil
	}
	if !hasID || hasResult == hasError {
		return errors.New("reply must contain exactly one of result or error and an id")
	}
	if isJSONNull(fields["id"]) {
		if !hasError {
			return errors.New("null id is only valid on error replies")
		}
	} else if err := validateID(fields["id"]); err != nil {
		return err
	}
	if hasError {
		// A reply's error must be the §4 object: code and message are required,
		// and data, when present, must carry a string appError. Rejecting a
		// malformed error here keeps it from surfacing as a zero-value daemon
		// error on some caller's request.
		var e struct {
			Code    *int            `json:"code"`
			Message *string         `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if isJSONNull(fields["error"]) {
			return errors.New("error must be an object")
		}
		if err := json.Unmarshal(fields["error"], &e); err != nil {
			return fmt.Errorf("invalid error object: %w", err)
		}
		if e.Code == nil || e.Message == nil {
			return errors.New("error object must contain code and message")
		}
		if len(e.Data) > 0 {
			var data struct {
				AppError *string `json:"appError"`
			}
			if err := json.Unmarshal(e.Data, &data); err != nil || data.AppError == nil {
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

// isJSONNull reports whether raw is the JSON literal null, ignoring
// surrounding whitespace.
func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
