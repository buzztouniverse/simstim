# The launcher: the hosts, one host's sessions, a peek at a session's screen,
# search across every host, and the quick-start line. Run by
# scripts/screenshots.sh against the demo, where nothing connects anywhere.
record start
wait "homelab" 10s
sleep 1500ms
send "\x1b[B"
sleep 500ms
send "\x1b[B"
sleep 900ms
send "\r"
sleep 1600ms
send "\x1b[B"
sleep 700ms
send " "
sleep 2200ms
send "\x1b"
sleep 900ms
send "/"
sleep 400ms
type "api" 140ms
sleep 1600ms
send "\x1b"
sleep 800ms
send "n"
sleep 500ms
type "codex ~/src/api fix-tests" 60ms
sleep 1800ms
send "\x1b"
sleep 600ms
record save docs/images/launcher.svg docs/images/launcher.cast 2500ms
