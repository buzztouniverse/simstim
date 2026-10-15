package term

// Catchup follows a client's place in a session's output, so that frames from
// a replay, a snapshot or a reconnect reach the terminal once and in order,
// and so that replayed history carries no query the terminal would answer.
type Catchup struct {
	// Offset is the offset after the last byte the terminal shows.
	Offset int64
	// Live is where replayed history ends and live output begins.
	Live int64
	san  Sanitizer
}

// History starts a new stretch of replayed history that ends at live, as
// after an attach.
func (c *Catchup) History(live int64) {
	c.Live = live
	c.san = Sanitizer{}
}

// Feed takes an output frame that starts at offset off. It returns the bytes
// to write to the terminal, and the frame's new bytes as they came, for a mode
// tracker. Both are empty when the terminal already shows the whole frame.
func (c *Catchup) Feed(off int64, data []byte) (show, fresh []byte) {
	end := off + int64(len(data))
	if end <= c.Offset {
		return nil, nil
	}
	if off < c.Offset {
		data = data[c.Offset-off:]
		off = c.Offset
	}
	show = data
	if off < c.Live {
		n := min(c.Live-off, int64(len(data)))
		hist := c.san.Feed(data[:n])
		if off+n >= c.Live {
			hist = append(hist, c.san.Flush()...)
		}
		hist = append(hist, data[n:]...)
		show = hist
	}
	c.Offset = end
	return show, data
}
