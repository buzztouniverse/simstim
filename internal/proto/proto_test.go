package proto

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	frames := []Frame{
		{Type: TInput, Payload: []byte("ls\r")},
		{Type: TPing, Payload: []byte{}},
		{Type: TOutput, Payload: EncodeOutput(42, []byte("\x1b[2Jhi"))},
	}
	for _, f := range frames {
		if err := w.Write(f.Type, f.Payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.WriteJSON(TResize, Resize{Cols: 80, Rows: 24}); err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(&buf)
	for _, want := range frames {
		got, err := Read(r)
		if err != nil {
			t.Fatal(err)
		}
		if got.Type != want.Type || !bytes.Equal(got.Payload, want.Payload) {
			t.Fatalf("got %v %q, want %v %q", got.Type, got.Payload, want.Type, want.Payload)
		}
	}
	got, err := Read(r)
	if err != nil || got.Type != TResize || !strings.Contains(string(got.Payload), `"cols":80`) {
		t.Fatalf("resize frame: %v %q %v", got.Type, got.Payload, err)
	}
	if _, err := Read(r); !errors.Is(err, io.EOF) {
		t.Fatalf("after the last frame: %v, want EOF", err)
	}
}

func TestReadRejectsBadFrames(t *testing.T) {
	header := func(n uint32, rest ...byte) []byte {
		b := binary.LittleEndian.AppendUint32(nil, n)
		return append(b, rest...)
	}
	for name, tc := range map[string]struct {
		in   []byte
		want error
	}{
		"empty":     {header(0), nil},
		"too big":   {header(MaxFrame + 1), ErrFrameTooBig},
		"truncated": {header(5, byte(TInput), 'a'), io.ErrUnexpectedEOF},
		"no body":   {header(3), io.ErrUnexpectedEOF},
		"short hdr": {[]byte{1, 0}, io.ErrUnexpectedEOF},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Read(bufio.NewReader(bytes.NewReader(tc.in)))
			if err == nil {
				t.Fatal("no error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestWriteRejectsTooBig(t *testing.T) {
	w := NewWriter(io.Discard)
	if err := w.Write(TOutput, make([]byte, MaxFrame)); !errors.Is(err, ErrFrameTooBig) {
		t.Fatalf("got %v", err)
	}
	if err := w.WriteJSON(TRequest, func() {}); err == nil {
		t.Fatal("a function marshalled")
	}
}

// Frames written from many goroutines never interleave.
func TestWriterIsSafeForConcurrentUse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				_ = w.Write(TOutput, bytes.Repeat([]byte{byte('a' + i)}, 100+i))
			}
		}()
	}
	wg.Wait()
	r := bufio.NewReader(&buf)
	for n := 0; ; n++ {
		f, err := Read(r)
		if errors.Is(err, io.EOF) {
			if n != 16*50 {
				t.Fatalf("read %d frames, want %d", n, 16*50)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(f.Payload, f.Payload[:1]) != len(f.Payload) {
			t.Fatalf("frame %d is mixed: %q", n, f.Payload)
		}
	}
}

func TestOutputAndChunkPayloads(t *testing.T) {
	off, data, err := DecodeOutput(EncodeOutput(1<<40, []byte("abc")))
	if err != nil || off != 1<<40 || string(data) != "abc" {
		t.Fatalf("output: %d %q %v", off, data, err)
	}
	id, data, err := DecodeChunk(EncodeChunk(7, []byte("png")))
	if err != nil || id != 7 || string(data) != "png" {
		t.Fatalf("chunk: %d %q %v", id, data, err)
	}
	if _, _, err := DecodeOutput([]byte{1, 2, 3}); err == nil {
		t.Error("a short output frame decoded")
	}
	if _, _, err := DecodeChunk(nil); err == nil {
		t.Error("a short chunk decoded")
	}
}

func TestHelloCheck(t *testing.T) {
	ok := Hello{Magic: Magic, Protocol: "1.3"}
	if err := ok.Check("1.0"); err != nil {
		t.Errorf("same major: %v", err)
	}
	for _, h := range []Hello{
		{Magic: "other", Protocol: "1.0"},
		{Magic: Magic, Protocol: "2.0"},
		{Magic: Magic},
	} {
		if err := h.Check("1.0"); err == nil {
			t.Errorf("%+v passed", h)
		}
	}
	e := &Error{Code: "no_session", Message: "no such session"}
	if e.Error() != "no such session" {
		t.Errorf("Error() = %q", e.Error())
	}
}

// Read never panics and never returns more than MaxFrame, whatever the peer
// sends.
func FuzzRead(f *testing.F) {
	var seed bytes.Buffer
	w := NewWriter(&seed)
	_ = w.Write(TInput, []byte("hi"))
	_ = w.WriteJSON(THello, Hello{Magic: Magic, Protocol: "1.0"})
	f.Add(seed.Bytes())
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 1})
	f.Fuzz(func(t *testing.T, in []byte) {
		r := bufio.NewReader(bytes.NewReader(in))
		for {
			fr, err := Read(r)
			if err != nil {
				return
			}
			if len(fr.Payload)+1 > MaxFrame {
				t.Fatalf("frame of %d bytes", len(fr.Payload))
			}
			if fr.Type == TOutput {
				_, _, _ = DecodeOutput(fr.Payload)
			}
		}
	})
}

func BenchmarkWriteOutput(b *testing.B) {
	w := NewWriter(io.Discard)
	data := bytes.Repeat([]byte("x"), 32<<10)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if err := w.Write(TOutput, EncodeOutput(int64(i)*int64(len(data)), data)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRead(b *testing.B) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	data := bytes.Repeat([]byte("x"), 32<<10)
	for range 64 {
		_ = w.Write(TOutput, EncodeOutput(0, data))
	}
	raw := buf.Bytes()
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		r := bufio.NewReaderSize(bytes.NewReader(raw), 64<<10)
		for {
			if _, err := Read(r); err != nil {
				break
			}
		}
	}
}
