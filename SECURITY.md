# Security policy

This tool runs AI agents with shell access on your hosts and carries your terminal traffic, so we take security reports seriously.

## Report a vulnerability

Do not open a public issue. Use GitHub's private reporting: **Security → Report a vulnerability** on this repository. We aim to answer within 7 days.

Include the version, the host and client systems, the steps to reproduce, and the impact you see.

## Supported versions

The project has no release yet. Once it does, only the latest minor release gets security fixes.

## Design rules that limit risk

- Nothing new listens on a host. The only way in is the host's own SSH server.
- There is no relay, no cloud service and no telemetry.
- The tool stores no secrets. Agent logins stay on each host.
- SSH agent forwarding is off by default.
- Host helpers are checked by SHA-256 before they are installed.
- Sockets and uploaded files are private to the user (modes 0600 and 0700).
- Ports forwarded to the laptop bind to `127.0.0.1` only.

[The architecture](docs/architecture.md#7-security-model) has the full threat model.
