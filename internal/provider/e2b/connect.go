package e2b

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// The Connect protocol, as much of it as envd's process service needs, with
// the JSON codec (https://connectrpc.com/docs/protocol/). A streaming body is
// a run of envelopes: a flags byte, a 4-byte big-endian length, then the
// message. The last envelope of a response has flagEndStream and carries the
// end-of-stream JSON. Bytes fields travel as base64, so the stream stays
// binary-safe.

const (
	flagEndStream = 0x02
	// maxEnvelope bounds one message, well above envd's output chunks.
	maxEnvelope   = 16 << 20
	contentUnary  = "application/json"
	contentStream = "application/connect+json"
)

// writeEnvelope writes one message.
func writeEnvelope(w io.Writer, flags byte, msg []byte) error {
	var hdr [5]byte
	hdr[0] = flags
	binary.BigEndian.PutUint32(hdr[1:], uint32(len(msg)))
	if _, err := w.Write(append(hdr[:], msg...)); err != nil {
		return fmt.Errorf("write a Connect envelope: %w", err)
	}
	return nil
}

// readEnvelope reads one message.
func readEnvelope(r io.Reader) (byte, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	n := binary.BigEndian.Uint32(hdr[1:])
	if n > maxEnvelope {
		return 0, nil, fmt.Errorf("a Connect message of %d bytes is too large", n)
	}
	msg := make([]byte, n)
	if _, err := io.ReadFull(r, msg); err != nil {
		return 0, nil, fmt.Errorf("read a Connect message: %w", err)
	}
	return hdr[0], msg, nil
}

// rpcError is a Connect error: {"code": "not_found", "message": "…"}.
type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	if e.Message == "" {
		return "envd: " + e.Code
	}
	return "envd: " + e.Code + ": " + e.Message
}

// endStream is the last message of a stream.
type endStream struct {
	Error *rpcError `json:"error"`
}

// endError reads the end-of-stream message; nil means the stream ended well.
func endError(msg []byte) error {
	var end endStream
	if err := json.Unmarshal(msg, &end); err != nil {
		return fmt.Errorf("read the end of a Connect stream: %w", err)
	}
	if end.Error != nil {
		return end.Error
	}
	return nil
}

// responseError turns a non-200 Connect response into an error. A proxy in
// front of envd may answer with plain text, which is kept as the message.
func responseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var e rpcError
	if json.Unmarshal(body, &e) == nil && e.Code != "" {
		return &statusError{Status: resp.StatusCode, Err: &e}
	}
	return &statusError{Status: resp.StatusCode, Err: errors.New(firstLine(string(body)))}
}

// statusError is an error with the HTTP status it came with.
type statusError struct {
	Status int
	Err    error
}

func (e *statusError) Error() string { return fmt.Sprintf("HTTP %d: %v", e.Status, e.Err) }
func (e *statusError) Unwrap() error { return e.Err }

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
