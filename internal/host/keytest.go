package host

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/term"
)

// KeyTest prints every key the terminal sends, as quoted bytes. Run inside a
// session, it proves that keys cross the whole path unchanged.
func KeyTest() error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("keytest needs a terminal")
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("put the terminal in raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, old) }()
	fmt.Print("\x1b[>31u")
	defer fmt.Print("\x1b[<u")
	fmt.Print("Key test on the host. Press keys; press q twice to quit.\r\n\r\n")
	buf := make([]byte, 4096)
	qs := 0
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}
		s := string(buf[:n])
		fmt.Printf("%s\r\n", strconv.Quote(s))
		if s == "q" || s == "\x1b[113u" {
			qs++
			if qs >= 2 {
				return nil
			}
		} else if s != "\x1b[113;1:3u" {
			qs = 0
		}
	}
}
