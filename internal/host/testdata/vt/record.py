#!/usr/bin/env python3
"""Records vim.bin: the output of a real full-screen program, vim, in a
100 by 30 pseudo-terminal, for the shadow terminal tests (vt_test.go).

It needs vim with its runtime files (Debian: vim). It edits gen.py, so the
recording holds nothing from the machine it ran on. Run it from this folder:

    python3 record.py
"""

import fcntl
import os
import pty
import select
import struct
import termios
import time

COLS, ROWS = 100, 30

# Keys to type, each after a pause in seconds.
KEYS = [
    (1.0, b"40j"),
    (0.5, b"/def \r"),
    (0.5, b":vsplit\r"),
    (0.5, b"\x17l"),
    (0.5, b"Gonew line, caf\xc3\xa9 \xe7\x95\x8c\x1b"),
    (0.5, b":set cursorline\r"),
    (0.5, b"gg"),
    (0.5, b""),
]


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    pid, fd = pty.fork()
    if pid == 0:
        fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))
        env = {"TERM": "xterm-256color", "LANG": "C.UTF-8", "PATH": "/usr/bin:/bin", "HOME": "/nonexistent"}
        os.chdir(here)
        os.execve("/usr/bin/vim", ["vim", "-u", "NONE", "-N", "-i", "NONE", "-n",
                                   "-c", "syntax on",
                                   "-c", "set number laststatus=2 background=dark ruler",
                                   "gen.py"], env)
    out = bytearray()

    def read_for(seconds):
        end = time.time() + seconds
        while time.time() < end:
            r, _, _ = select.select([fd], [], [], max(0.0, end - time.time()))
            if r:
                try:
                    out.extend(os.read(fd, 65536))
                except OSError:
                    return

    for pause, keys in KEYS:
        read_for(pause)
        if keys:
            os.write(fd, keys)
    os.kill(pid, 9)
    os.waitpid(pid, 0)
    with open(os.path.join(here, "vim.bin"), "wb") as f:
        f.write(out)
    print("wrote vim.bin, %d bytes" % len(out))


if __name__ == "__main__":
    main()
