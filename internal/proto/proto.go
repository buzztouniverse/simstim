// Package proto is the wire protocol between the client, the host bridge
// and the session holders.
//
// A frame is a little-endian uint32 length (of type byte plus payload), one type
// byte, then the payload. Control payloads are JSON. Input and output payloads are
// raw bytes. Types are append-only: a peer ignores types it does not know.
package proto

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/iAmLakshya/simstim/internal/brand"
)

// Magic identifies a peer in the hello frame.
const Magic = brand.Name

// MaxFrame is the largest frame either side accepts.
const MaxFrame = 4 << 20

// Type is the kind of a frame.
type Type uint8

// Frame types.
const (
	THello    Type = 0x01
	TError    Type = 0x02
	TPing     Type = 0x03
	TPong     Type = 0x04
	TRequest  Type = 0x10
	TResponse Type = 0x11
	TAttach   Type = 0x20
	TAttached Type = 0x21
	TDetach   Type = 0x22
	TInput    Type = 0x23
	TOutput   Type = 0x24
	TResize   Type = 0x25
	TEnded    Type = 0x26
	TEvent    Type = 0x30

	TUploadBegin Type = 0x40
	TUploadChunk Type = 0x41
	TUploadEnd   Type = 0x42
)

// Frame is one decoded frame.
type Frame struct {
	Type    Type
	Payload []byte
}

// ErrFrameTooBig is returned for a frame over MaxFrame.
var ErrFrameTooBig = errors.New("proto: frame too big")

// Read reads one frame.
func Read(r *bufio.Reader) (Frame, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Frame{}, err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n == 0 {
		return Frame{}, errors.New("proto: empty frame")
	}
	if n > MaxFrame {
		return Frame{}, ErrFrameTooBig
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return Frame{}, err
	}
	return Frame{Type: Type(buf[0]), Payload: buf[1:]}, nil
}

// Writer writes whole frames. It is safe for concurrent use.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewWriter wraps w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write writes one frame in a single call to the underlying writer.
func (w *Writer) Write(t Type, payload []byte) error {
	if len(payload)+1 > MaxFrame {
		return ErrFrameTooBig
	}
	buf := make([]byte, 5+len(payload))
	binary.LittleEndian.PutUint32(buf, uint32(len(payload)+1))
	buf[4] = byte(t)
	copy(buf[5:], payload)
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.w.Write(buf)
	return err
}

// WriteJSON marshals v and writes it as one frame.
func (w *Writer) WriteJSON(t Type, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.Write(t, b)
}

// Output frames carry the absolute offset of their first byte.

// EncodeOutput builds an output payload.
func EncodeOutput(offset int64, data []byte) []byte {
	buf := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint64(buf, uint64(offset))
	copy(buf[8:], data)
	return buf
}

// DecodeOutput splits an output payload.
func DecodeOutput(p []byte) (int64, []byte, error) {
	if len(p) < 8 {
		return 0, nil, errors.New("proto: short output frame")
	}
	return int64(binary.LittleEndian.Uint64(p)), p[8:], nil
}

// Upload chunks carry the upload ID first.

// EncodeChunk builds an upload chunk payload.
func EncodeChunk(id uint64, data []byte) []byte {
	buf := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint64(buf, id)
	copy(buf[8:], data)
	return buf
}

// DecodeChunk splits an upload chunk payload.
func DecodeChunk(p []byte) (uint64, []byte, error) {
	if len(p) < 8 {
		return 0, nil, errors.New("proto: short upload chunk")
	}
	return binary.LittleEndian.Uint64(p), p[8:], nil
}

// Hello is the first frame each way.
type Hello struct {
	Magic    string   `json:"magic"`
	Protocol string   `json:"protocol"`
	Version  string   `json:"version"`
	Role     string   `json:"role"`
	Features []string `json:"features,omitempty"`
}

// Check reports whether h is a compatible peer.
func (h Hello) Check(major string) error {
	if h.Magic != Magic {
		return fmt.Errorf("proto: not a %s peer (magic %q)", brand.Name, h.Magic)
	}
	if len(h.Protocol) == 0 || h.Protocol[0] != major[0] {
		return fmt.Errorf("proto: incompatible protocol %s, want %s.x", h.Protocol, major[:1])
	}
	return nil
}

// Error is a user-facing error with a suggested next step.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Request asks the bridge or a holder to do something.
type Request struct {
	ID   uint64          `json:"id"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
}

// Response answers a request with the same ID.
type Response struct {
	ID     uint64          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Attach asks to attach to a session.
type Attach struct {
	Session string `json:"session"`
	Cols    int    `json:"cols"`
	Rows    int    `json:"rows"`
	XPx     int    `json:"xpx,omitempty"`
	YPx     int    `json:"ypx,omitempty"`
	Client  string `json:"client"`
	// LastOffset is the offset after the last byte this client's terminal shows,
	// or -1 when the terminal has none of this session's output.
	LastOffset int64 `json:"last_offset"`
}

// Attached confirms an attach.
type Attached struct {
	Session string `json:"session"`
	// Mode is "replay" (bytes from LastOffset follow) or "full" (the whole ring follows).
	Mode string `json:"mode"`
	// From is the offset of the first byte that follows.
	From int64 `json:"from"`
	// Live is the offset where replayed history ends and live output begins.
	Live int64 `json:"live"`
	// Exited is true when the agent has already exited.
	Exited bool `json:"exited,omitempty"`
}

// Resize changes the session's window size.
type Resize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
	XPx  int `json:"xpx,omitempty"`
	YPx  int `json:"ypx,omitempty"`
}

// Ended reports that the agent exited.
type Ended struct {
	ExitCode int    `json:"exit_code"`
	Signal   string `json:"signal,omitempty"`
	OOM      bool   `json:"oom,omitempty"`
}

// UploadBegin starts an upload of a file into a session's paste folder.
type UploadBegin struct {
	ID      uint64 `json:"id"`
	Session string `json:"session"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
}

// UploadEnd finishes an upload. The bridge answers with a Response carrying UploadResult.
type UploadEnd struct {
	ID     uint64 `json:"id"`
	SHA256 string `json:"sha256"`
}

// UploadResult is the result of a finished upload.
type UploadResult struct {
	Path string `json:"path"`
}
