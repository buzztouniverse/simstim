# Joining a session: Enter opens it, the agent owns the whole terminal, and
# Ctrl+\ detaches back to the launcher while the session keeps running.
record start
wait "homelab" 10s
sleep 1200ms
send "\x1b[B"
sleep 400ms
send "\x1b[B"
sleep 600ms
send "\r"
sleep 1500ms
send "\r"
sleep 3500ms
type "run the tests again and fix what fails" 55ms
sleep 1200ms
send "\r"
sleep 2500ms
send "\x1c"
sleep 2500ms
record save docs/images/attach.svg docs/images/attach.cast 2500ms
