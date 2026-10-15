# Testing

Every change keeps the tests green, and every bug fix adds the test that would have caught it.

| Command | What it runs | Where |
| --- | --- | --- |
| `just test` | Unit tests with the race detector | Anywhere |
| `go test ./internal/tui -run TestGolden -update` | Rewrites the golden screens after an intended change; review the diff | Anywhere |
| `just fuzz` | Fuzzes the input filter for 30 s (`just fuzz 5m` for longer) | Anywhere |
| `just lint` | `go vet`, golangci-lint, the name check and the private-detail check | Anywhere |
| `just e2e` | The end-to-end suite against a real host | Needs a test host |
| `just screenshots` | The README images, drawn from `demo` | Anywhere |
| `just repro` | Two release builds from empty caches; every binary must match | Needs GoReleaser and Zig |
| `just test-install` | `install.sh` against a local, signed snapshot release | Needs GoReleaser and Zig |

## Unit and golden tests

- **Screens** are golden files in `internal/tui/testdata`, at 60×20, 80×24 and 120×36, with motion off. A golden test fails on any change of text or layout. Run it with `-update` only when the change is meant, and read the diff.
- **Colour** is tested separately in `internal/tui/theme_test.go`: each screen is rendered as a truecolour, 256-colour, 16-colour, `NO_COLOR` and no-terminal profile would receive it, and every fixed colour must keep a contrast of 3:1 on dark and light backgrounds.
- **Model tests** (`internal/tui/model_test.go`) drive the launcher with keys against a fake backend, and check what it asks the backend to do.
- **The attach path** is byte-exact: any change to it needs a test that proves the bytes are unchanged.

## End-to-end tests

`test/e2e/run.py` drives the real client in a pseudo-terminal, as a person in Ghostty would, against a real Linux host.

1. Use a disposable Linux VM with systemd. The suite starts and deletes sessions named `e2e-*`, installs the helper, and briefly fills memory to test the cap. Do not point it at a machine you care about.
2. Make one with `simstim host create lima e2e` ([how to](../how-to/create-a-host.md)), or add one you reach with `ssh`: `simstim host add <name> <ssh-target>`.
3. Run `E2E_HOST=<name> just e2e`. It prints `PASS` or `FAIL` for each check and a total.
4. Delete a created host afterwards with `simstim host destroy e2e`.

A container made with `host create docker` runs the suite too, over `docker exec` instead of SSH. It has no systemd, so the memory cap check is skipped. CI runs the suite this way on every change (the `docker-host` job).

It covers start, attach and detach, resize, byte-exact output, replay after a dropped link, a simulated laptop sleep, the memory cap, snapshots after the ring wraps, a crashed shadow terminal, image paste and file drops, worktrees, ports, rename and delete. Host reboots are tested by hand, with the host owner's agreement.

## Seeing a screen

`test/ttyshot` runs a program in a pseudo-terminal, sends keys from a script, and prints the screen it ends on, or draws it as SVG:

```sh
go run ./test/ttyshot -script keys.tty -- ./bin/<name> demo
```

`just screenshots` uses it to draw the README's images.
