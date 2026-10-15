"""Drive the client in a real pseudo-terminal, like a person in Ghostty would."""
import os, pty, select, struct, fcntl, termios, time, sys, re, signal

class Term:
    def __init__(self, argv, cols=120, rows=36, env=None):
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            e = dict(os.environ); e.update(env or {})
            os.execvpe(argv[0], argv, e)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
        self.buf = b""
    def read(self, timeout=0.5):
        end = time.time() + timeout
        while time.time() < end:
            r, _, _ = select.select([self.fd], [], [], max(0, end - time.time()))
            if not r: break
            try: data = os.read(self.fd, 65536)
            except OSError: break
            if not data: break
            self.buf += data
        return self.buf
    def wait_for(self, pattern, timeout=20):
        end = time.time() + timeout
        while time.time() < end:
            self.read(0.2)
            if re.search(pattern, self.buf): return True
        return False
    def send(self, b):
        os.write(self.fd, b)
    def resize(self, cols, rows):
        # The kernel sends SIGWINCH to the client, as a terminal would.
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
    def exit_code(self, timeout=10):
        end = time.time() + timeout
        while time.time() < end:
            pid, st = os.waitpid(self.pid, os.WNOHANG)
            if pid: return os.waitstatus_to_exitcode(st)
            self.read(0.1)
        return None
