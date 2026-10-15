---
status: proposed
date: 2026-10-15
---

# 0013 Host providers: Lima and E2B first, over a byte-stream transport

**Status:** Proposed — awaiting the owner's yes at Gate 1.

## Context

Today simstim knows nothing about providers. A host is anything the system `ssh` can reach, and a user script can start and stop it ([architecture, 8](../architecture.md#8-providers)). The project now wants `host create <provider>` and `host destroy`: one command that creates a host, installs the helper and connects, with cloud hosts that stop when idle and wake on attach.

The rules stand: no relay, no telemetry, no new listening port on a host without a decision record and the owner's yes, and no secrets stored by simstim.

The research is in [Sandboxes and cloud hosts](../research/sandboxes.md). Its main finding: the helper's protocol needs only a two-way, binary-safe byte stream to one process. `ssh -T` is one; so are `docker exec -i` and E2B's envd process API. A provider's own authenticated exec endpoint is a door that already exists, so a provider without SSH can work with no new port. One provider must also be a local VM or container, so the whole flow can be proved live on a Mac or a Linux box without an account.

## Options

**Which providers first:**

1. Lima only.
2. Lima and one VM cloud (GCE or Hetzner), over SSH only.
3. Lima over SSH, and E2B over its exec API, with Docker as the local test bed for that transport.
4. Lima, and Morph Cloud, an SSH-native sandbox.
5. No built-in providers: extend the provider scripts to `create` and `destroy`, and ship example scripts.

**Transport:**

1. SSH only. A provider without SSH gets an sshd, which on E2B means a new public websocat port.
2. A transport seam: SSH by default, or a byte stream from the provider's own exec API.

**Lifecycle code:**

1. Small built-in adapters: plain `net/http` against each REST API, or the provider's local CLI (`limactl`, `docker`).
2. The providers' official SDKs.

**Idle stop:**

1. The client stops the host when the user detaches.
2. Inside the guest for VMs (power off when idle); a lease that the client renews for sandboxes.
3. Only what each provider offers natively.

**Credentials:**

1. simstim keeps provider tokens in its config.
2. simstim reads them when needed, from the user's environment, the provider CLI's own login, or a command the user names, and never writes them.

## Decision

Option 3 for providers, and option 2 for transport, idle stop and credentials. Option 1 for lifecycle code.

**Providers.**

- **Lima, over SSH.** Free, no account, macOS and Linux, and a real VM with systemd and sshd, so it runs the cloud-VM code path unchanged.
- **E2B, over envd.** No SSH and no new port; pause keeps running sessions and costs nothing; the server is Apache-2.0 and self-hostable; the free credit pays for a live proof.
- **Docker, over `docker exec -i`, marked experimental.** It proves the second transport live, in CI and on any machine with a Docker engine, without an E2B account.
- **Later, in this order:** Morph, GCE, Hetzner (create and destroy only), EC2. Coder becomes a documented SSH recipe, not a provider.
- **Not supported:** Modal and Codespaces (hard lifetime caps), UTM and Colima.

**Transport.** One small interface in the client: start a remote command, then give a stdin writer, a stdout reader and a wait.

- `ssh`: today's path, unchanged.
- `docker`: shells out to `docker exec -i <container> …`.
- `e2b`: a Go client for envd's process service, built from its Apache-2.0 proto. `Start` carries stdout; `StreamInput`, or ordered `SendInput` calls, carry stdin. It sends no `Connect-Timeout-Ms` header, so envd sets no deadline. It tags each bridge, and kills the old one on reconnect.
- The helper protocol, the bootstrap script (`uname -sm`, then `cat > f; sha256sum -c; chmod; mv`) and the handshake do not change.
- The bridge exits when the client's pings stop for 30 s. On E2B a dropped request does not end the process.
- Port forwarding needs an SSH master, so `forward` stays SSH-only until RPC tunnelling exists.

**Create and bootstrap.** `host create` counts as consent to install the helper, but still shows what it installs and where.

- **Lima.**
  1. `limactl create --plain --cpus 2 --memory 4 --disk 40 template:ubuntu-lts`, then `limactl start`. `--plain` stops the home-folder mount and the forwarding of every guest port to the laptop.
  2. The normal host add, through Lima's generated `ssh.config`. simstim's generated wrapper includes it; `~/.ssh/config` is never edited.
  3. Linger on, with `sudo`.
- **E2B.**
  1. Make sure the user's team has a template with 2 vCPU and 4 GiB, built once. CPU and memory belong to the template, not to the sandbox.
  2. Create the sandbox with `autoPause: true`, `autoResume` off and `network.allowPublicTraffic: false`, labelled with the host's name in `metadata`.
  3. Probe, upload and hand-shake over envd.
- **Docker.** A container with `--init`, running `sleep infinity`. The helper uses its `setsid` path there.

**Idle stop.**

- **VMs (Lima now; GCE and EC2 later).** A systemd user timer runs a helper check, so no daemon is added. When no client is connected and no session is working for N minutes, the check runs `sudo -n systemctl poweroff`. N is 30 by default, and `never` turns it off. It needs no credential on the host. Without passwordless sudo there is no idle stop, and `doctor` says so.
- **Sandboxes (E2B now; Morph later).** While any client is connected, it renews the sandbox's timeout every minute: 60 minutes while any session is working, 15 when none is. When the laptop sleeps or closes, the lease runs out and the sandbox pauses with its memory. Both times can be changed.
- **Hetzner** gets no idle stop at first, because a stopped server is billed in full.

**Wake on attach.** This extends today's wake. The client asks the provider for the host's state first. If the host is stopped or paused, the client starts or resumes it (`limactl start`, `POST /v2/sandboxes/{id}/connect`), waits, reads the address again, then dials.

**Destroy.** `host destroy` asks first. It refuses while a session is working, unless `--force`. It deletes the VM or sandbox, then removes the host from `hosts.toml` and from the generated SSH config.

**Credentials.**

- For E2B, the token comes from `E2B_API_KEY`, then the E2B CLI's own login file, then a `token_command` the user sets in `hosts.toml`, for example a keychain read.
- simstim never writes the token, logs it, or sends it to a host.
- envd's per-sandbox token is fetched from the API on each connect and kept in memory only.
- Lima and Docker need no credential.

## Consequences

- Good, because `host create lima` proves create, bootstrap, connect, idle stop, wake and destroy live, with no account. The Docker path can run in CI.
- Good, because nothing new listens on any host. Lima's sshd is the VM's ordinary door. envd and the Docker socket belong to their providers.
- Good, because E2B's pause freezes sessions instead of killing them, and costs nothing while paused.
- Good, because the transport seam makes later providers cheap: Incus, Apple's `container` and Morph's lifecycle all fit it.
- Good, because simstim still stores no secrets.
- Bad, because a second transport means more code and tests. Whether E2B's edge carries a long client stream is UNVERIFIED. The fallback costs one round trip per stdin chunk, so a large image paste is slow.
- Bad, because E2B's timeout is a lease, not an idle timer. An agent still working when the laptop sleeps is frozen when the lease ends. Its open request to its model API fails on resume. On the free plan every run also stops after 1 hour.
- Bad, because E2B needs a template build in the user's account on first use, since agents want 4 GB or more.
- Bad, because E2B's edge carries every byte between the laptop and the sandbox. It is the provider's own access path, as IAP is on GCP, and E2B already runs the VM.
- Bad, because `forward` does not work on exec-transport hosts yet.
- Bad, because Docker's default seccomp profile blocks bubblewrap. Agents' own sandboxes fail there, so Docker stays a test bed.
- Bad, because idle stop on a VM needs passwordless sudo.
- After the owner's yes, the E2B proof needs an E2B account and key, made by the owner. This research made none.
