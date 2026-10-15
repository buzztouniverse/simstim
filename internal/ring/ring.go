// Package ring is a fixed-size byte ring that remembers absolute offsets.
//
// Every byte a session ever wrote has an offset. The ring keeps the newest
// Size bytes, so a client that knows its last offset can catch up exactly.
package ring

// Ring holds the newest bytes of a stream. It is not safe for concurrent use.
type Ring struct {
	buf []byte
	end int64 // total bytes ever written
}

// New makes a ring that keeps size bytes.
func New(size int) *Ring {
	if size <= 0 {
		panic("ring: size must be positive")
	}
	return &Ring{buf: make([]byte, size)}
}

// End is the offset after the newest byte.
func (r *Ring) End() int64 { return r.end }

// Start is the offset of the oldest byte still held.
func (r *Ring) Start() int64 {
	if s := r.end - int64(len(r.buf)); s > 0 {
		return s
	}
	return 0
}

// Write appends p.
func (r *Ring) Write(p []byte) {
	n := len(r.buf)
	if len(p) >= n {
		// Only the tail fits.
		r.end += int64(len(p) - n)
		p = p[len(p)-n:]
	}
	pos := int(r.end % int64(n))
	c := copy(r.buf[pos:], p)
	if c < len(p) {
		copy(r.buf, p[c:])
	}
	r.end += int64(len(p))
}

// Has reports whether off is between Start and End.
func (r *Ring) Has(off int64) bool { return off >= r.Start() && off <= r.end }

// ReadFrom returns a copy of the bytes from off to End.
// ok is false when off is no longer (or not yet) in the ring.
func (r *Ring) ReadFrom(off int64) (data []byte, ok bool) {
	if !r.Has(off) {
		return nil, false
	}
	n := int(r.end - off)
	out := make([]byte, n)
	size := int64(len(r.buf))
	pos := int(off % size)
	c := copy(out, r.buf[pos:])
	if c < n {
		copy(out[c:], r.buf[:n-c])
	}
	return out, true
}

// Tail returns a copy of the newest n bytes (or fewer), and their start offset.
func (r *Ring) Tail(n int) ([]byte, int64) {
	start := r.end - int64(n)
	if start < r.Start() {
		start = r.Start()
	}
	data, _ := r.ReadFrom(start)
	return data, start
}
