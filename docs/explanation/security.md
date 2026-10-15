# Security model

What the tool does to keep your hosts and your laptop safe, and what it leaves to you. It is part of [the architecture](architecture.md); to report a problem, see [SECURITY.md](../../SECURITY.md).

**Rules:**

- **Nothing new listens on a host.** sshd is the only door. There is no relay and no telemetry.
- **simstim stores no secrets.** Agent logins live on the host, scoped to that host. Fine-grained GitHub tokens should be per host.
- **SSH agent forwarding is off by default.** If a user turns it on, the stable `SSH_AUTH_SOCK` link makes it work across reconnects. The risk is written in the docs.
- **Helpers are released signed,** and checked by SHA-256 before install.
- **Sockets and staged files are private:** 0600 and 0700, owner checked.
- **Forwarded ports (v2) bind to `127.0.0.1` only.** Nothing is ever public.
- **Untrusted host mode** filters OSC 52, OSC 8 and report sequences from the host.

**Threat model, in short:**

- **A compromised host** can see everything its agents can see. It cannot reach the laptop beyond the escape sequences it prints, and untrusted mode filters those.
- **A compromised laptop** can drive every host it has keys for, as it already can through `ssh`.
