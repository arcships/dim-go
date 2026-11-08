package appserver

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// chunkReader returns data in a repeating pattern of chunk sizes so a test can
// put frame boundaries at arbitrary positions without sleeping.
type chunkReader struct {
	data  []byte
	sizes []int
	pos   int
	next  int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	size := r.sizes[r.next%len(r.sizes)]
	r.next++
	if size < 1 {
		size = 1
	}
	if size > len(p) {
		size = len(p)
	}
	if remain := len(r.data) - r.pos; size > remain {
		size = remain
	}
	copy(p, r.data[r.pos:r.pos+size])
	r.pos += size
	return size, nil
}

// maxSizedFrame builds a syntactically valid JSON frame whose length is exactly
// MaxFrameSize bytes.
func maxSizedFrame(t *testing.T) []byte {
	t.Helper()
	const prefix = `{"jsonrpc":"2.0","method":"x","params":{"pad":"`
	const suffix = `"}}`
	padding := MaxFrameSize - len(prefix) - len(suffix)
	if padding < 0 {
		t.Fatalf("MaxFrameSize %d is too small for the fixture", MaxFrameSize)
	}
	frame := make([]byte, 0, MaxFrameSize)
	frame = append(frame, prefix...)
	frame = append(frame, bytes.Repeat([]byte("a"), padding)...)
	frame = append(frame, suffix...)
	if len(frame) != MaxFrameSize {
		t.Fatalf("fixture length = %d, want %d", len(frame), MaxFrameSize)
	}
	return frame
}

func TestFrameReaderAcceptsExactMaxFrameSize(t *testing.T) {
	frame := maxSizedFrame(t)
	input := append(append(make([]byte, 0, len(frame)+1), frame...), '\n')
	reader := NewFrameReader(&chunkReader{data: input, sizes: []int{4093, 4096, 17, 5000}})
	got, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if len(got) != MaxFrameSize {
		t.Fatalf("frame length = %d, want %d", len(got), MaxFrameSize)
	}
	if !bytes.Equal(got, frame) {
		t.Fatal("frame bytes changed while reading")
	}
	if _, err := reader.ReadFrame(); !errors.Is(err, io.EOF) {
		t.Fatalf("second read = %v, want EOF", err)
	}
}

func TestFrameReaderRejectsMaxFrameSizePlusOne(t *testing.T) {
	oversized := append(maxSizedFrame(t), 'x')
	if len(oversized) != MaxFrameSize+1 {
		t.Fatalf("fixture length = %d, want %d", len(oversized), MaxFrameSize+1)
	}
	input := append(oversized, '\n')
	_, err := NewFrameReader(&chunkReader{data: input, sizes: []int{4096, 7}}).ReadFrame()
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("error = %v, want ErrFrameTooLarge", err)
	}
}

func TestFrameReaderFrameSplitFixtures(t *testing.T) {
	small := [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"ping"}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"result":null}`),
	}
	var smallInput []byte
	smallInput = append(smallInput, '\n')       // blank line
	smallInput = append(smallInput, '\r', '\n') // CRLF blank line
	smallInput = append(smallInput, small[0]...)
	smallInput = append(smallInput, '\r', '\n')
	smallInput = append(smallInput, '\n')
	smallInput = append(smallInput, small[1]...)
	smallInput = append(smallInput, '\n')

	maxFrame := maxSizedFrame(t)
	maxInput := append(append(make([]byte, 0, len(maxFrame)+1), maxFrame...), '\n')

	cases := []struct {
		name  string
		input []byte
		sizes []int
		want  [][]byte
	}{
		{"blank lines and CRLF with one-byte chunks", smallInput, []int{1}, small},
		{"prime chunk sizes", smallInput, []int{2, 3, 5, 7, 11}, small},
		{"bufio buffer boundaries", smallInput, []int{4095, 4096, 4097}, small},
		{"max frame split across chunks", maxInput, []int{4093, 4096, 17, 5000, 1}, [][]byte{maxFrame}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewFrameReader(&chunkReader{data: tc.input, sizes: tc.sizes})
			var got [][]byte
			for {
				frame, err := reader.ReadFrame()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatalf("ReadFrame: %v", err)
				}
				got = append(got, frame)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("frames = %d, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if !bytes.Equal(got[i], tc.want[i]) {
					t.Fatalf("frame %d differs: got %d bytes, want %d", i, len(got[i]), len(tc.want[i]))
				}
			}
		})
	}
}
