"""End-to-end tests against a real host. Run with `just e2e`.

Needs: a built client, and a host already added with `host add`.
  E2E_HOST        the host's name in the client (required)
The product name is read from internal/brand, so nothing here spells it.
"""
import collections, json, os, re, subprocess, sys, time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, HERE)
from harness import Term

NAME = re.search(r'^const Name = "(.*)"', open(os.path.join(ROOT, "internal/brand/brand.go")).read(), re.M).group(1)
cli = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, "bin", NAME))
HOST = os.environ.get("E2E_HOST")
if not HOST:
    sys.exit("set E2E_HOST to the name of a host added with `host add`")
WRAPPER = os.path.join(os.environ.get("XDG_CONFIG_HOME", os.path.expanduser("~/.config")), NAME, "ssh", "config-" + HOST)
HELPER = "~/.local/share/%s/bin/current/%s-host" % (NAME, NAME)
env = {"TERM": "xterm-ghostty", "TERM_PROGRAM": "ghostty", "COLORTERM": "truecolor"}
results = []

def ok(c, m):
    results.append(bool(c)); print(("PASS " if c else "FAIL ") + m); return c

def host_ssh(cmd, data=None):
    import tomllib
    hosts = tomllib.load(open(os.path.join(os.path.dirname(os.path.dirname(WRAPPER)), "hosts.toml"), "rb"))["host"]
    target = [h for h in hosts if h["name"] == HOST][0]["ssh"]
    return subprocess.run(["ssh", "-F", WRAPPER, "-o", "BatchMode=yes", target, cmd], input=data, capture_output=True, check=True)

def start(name, *argv):
    t = Term([cli, "new", HOST, "--name", name, "--detach", "--"] + list(argv), env=env)
    t.exit_code(60); m = re.search(rb"Started [^/]+/(\w+)", t.buf)
    if not m: print(t.buf.decode(errors="replace"))
    return m.group(1).decode() if m else None

created = []

# --- Attach, detach, reattach ---
# 1. Start a detached shell session.
t = Term([cli, "new", HOST, "--agent", "shell", "--name", "e2e-attach", "--detach"], env=env)
code = t.exit_code(60); out = t.buf.decode(errors="replace")
m = re.search(r"Started [^/]+/(\w+)", out)
ok(code == 0 and m, "new --detach: " + out.strip().splitlines()[-1])
sid = m.group(1); created.append(sid)
# 2. Attach, type, see the output.
t = Term([cli, "attach", HOST + "/" + sid], env=env)
ok(t.wait_for(rb"\$ ", 30), "attach shows the shell prompt")
t.send(b"echo MARK-$((40+2))\r")
ok(t.wait_for(rb"MARK-42", 10), "typed command runs on the host and its output comes back")
# 3. Detach with Ctrl+\ (legacy byte), wait for the double-tap window.
t.buf = b""
t.send(b"\x1c")
code = t.exit_code(10)
out = t.buf.decode(errors="replace")
ok(code == 0 and "Detached" in out, "Ctrl+\\ detaches and the client exits 0")
ok("\x1bc" not in out, "no RIS (ESC c) is ever sent to the terminal")
# 4. The session kept running: print in the background, then reattach (new client, full replay).
t = Term([cli, "attach", HOST + "/" + sid], env=env)
ok(t.wait_for(rb"MARK-42", 30), "reattach from a new client replays earlier output")
t.send(b"for i in 1 2 3; do echo LINE-$i; done\r")
ok(t.wait_for(rb"LINE-3", 10), "session still interactive after reattach")
# 5. Double Ctrl+\ sends the key to the session instead of detaching.
t.send(b"cat -v\r"); time.sleep(0.5)
t.buf = b""
t.send(b"\x1c\x1c"); time.sleep(0.8)
t.read(0.5)
ok(t.exit_code(0.5) is None, "double Ctrl+\\ does not detach")
t.send(b"\x04")  # end cat
time.sleep(0.5)
t.send(b"\x1c"); code = t.exit_code(10)
ok(code == 0, "single Ctrl+\\ then detaches")

# --- Reliability and fidelity ---
# A. Reconnect after the connection drops: no gap and no duplicate.
sid = start("e2e-counter", "/bin/bash", "-c", "for i in $(seq 1 400); do echo N$i; sleep 0.05; done; sleep 600")
t = Term([cli, "attach", HOST + "/" + sid], env=env)
ok(t.wait_for(rb"N20\r", 30), "counter running")
# Kill the client's bridge ssh (as a network drop would), and the shared master.
subprocess.run(["pkill", "-f", NAME + "-host bridge"], check=False)
subprocess.run(["pkill", "-f", "ControlPath=.*" + NAME + "/cm"], check=False)
t0 = time.time()
ok(t.wait_for(rb"N400\r", 60), "client reconnected by itself and kept receiving (%.1fs)" % (time.time() - t0))
nums = [int(x) for x in re.findall(rb"N(\d+)\r", t.buf)]
c = collections.Counter(nums)
dups = [n for n, k in c.items() if k > 1]
missing = [n for n in range(1, 401) if n not in c]
ok(not missing, "no gap after reconnect (missing: %s)" % missing[:10])
ok(not dups, "no duplicate after reconnect (dups: %s)" % dups[:10])
t.send(b"\x1c"); t.exit_code(10)

# B. Kitty keyboard sequences reach the host unchanged.
sid2 = start("e2e-keys", HELPER, "keytest")
t = Term([cli, "attach", HOST + "/" + sid2], env=env)
ok(t.wait_for(rb"Press keys", 30), "host key tester running in a session")
ok(b"\x1b[>31u" in t.buf, "kitty keyboard push reached the local terminal")
keys = [b"\x1b[13;2u", b"\x1b[112;3u", b"\x1b[98;5u", b"\x1b[27u", b"\x1b[99;5:3u"]
for k in keys:
    t.send(k); time.sleep(0.3)
t.read(1)
for k in keys:
    ok(repr(k.decode())[1:-1].encode() in t.buf.replace(b'\\x1b', b'\\x1b') or k.decode().encode('unicode_escape') in t.buf, "key %r arrived unchanged" % k)
t.send(b"q"); time.sleep(0.2); t.send(b"q")
time.sleep(1); t.read(1)

# C. Escape sequences pass byte for byte.
seq = (b"\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\ "
       b"\x1b[38;2;255;100;0mtruecolor\x1b[0m \x1b[4:3mcurly\x1b[0m "
       b"\x1b]777;notify;Title;Body\x1b\\\x1b]52;c;aGVsbG8=\x1b\\\x1b[?2026h\x1b[?2026l END")
host_ssh("cat > ~/e2e-seq.bin", seq)
sid3 = start("e2e-bytes", "/bin/bash", "-c", "cat ~/e2e-seq.bin; sleep 600")
t = Term([cli, "attach", HOST + "/" + sid3], env=env)
ok(t.wait_for(rb" END", 30), "escape test output arrived")
ok(seq in t.buf, "OSC 8, truecolor, styled underline, OSC 777, OSC 52 and sync output are byte-identical")
t.send(b"\x1c"); t.exit_code(10)

# D. A dropped local file is uploaded and its remote path pasted.
local = os.path.join(HERE, ".drop-test.txt")
open(local, "w").write("dropped file content\n")
sid4 = start("e2e-drop", "/bin/bash", "--norc", "-i")
t = Term([cli, "attach", HOST + "/" + sid4], env=env)
time.sleep(2); t.read(1)
t.send(b"bind 'set enable-bracketed-paste on' 2>/dev/null; printf '\\033[?2004h'; read -r P; cat $P\r"); time.sleep(1)
t.send(b"\x1b[200~" + local.encode() + b"\x1b[201~\r")
ok(t.wait_for(rb"/paste/[0-9a-f]{12}-drop-test.txt", 20), "dropped file uploaded and its remote path pasted")
t.send(b"\x1c"); t.exit_code(10)

# E. Killing the client leaves the session running.
t = Term([cli, "attach", HOST + "/" + sid4], env=env)
time.sleep(2)
os.kill(t.pid, 9)
time.sleep(1)
ls = subprocess.run([cli, "ls", "--json"], capture_output=True, text=True).stdout
ok('"id": "%s"' % sid4 in ls and '"alive": true' in ls, "session survives a killed client")

# F. Memory cap: a hog is OOM-killed and marked; another session is unaffected.
sid5 = start("e2e-oom", "/bin/bash", "-c", "head -c 3G /dev/zero | tail > /dev/null; sleep 600")
time.sleep(40)
ls = subprocess.run([cli, "ls", "--json"], capture_output=True, text=True).stdout
rows = {s["id"]: s for h in json.loads(ls) for s in (h["sessions"] or [])}
r5 = rows.get(sid5, {})
ok(rows.get(sid4, {}).get("alive"), "other sessions keep running")
print("oom session:", {k: r5.get(k) for k in ("state", "reason", "memory_current", "memory_max")})
ok(r5.get("oom_killed") is True, "the over-the-cap process was stopped and the session marked (reason: %s)" % r5.get("reason"))
created += [sid, sid2, sid3, sid4, sid5]

# Clean up.
for s in created:
    subprocess.run([cli, "kill", HOST + "/" + s], capture_output=True)
try:
    host_ssh("rm -f ~/e2e-seq.bin")
    os.remove(local)
except Exception:
    pass
print("RESULT %d/%d passed" % (sum(results), len(results)))
sys.exit(0 if all(results) else 1)
