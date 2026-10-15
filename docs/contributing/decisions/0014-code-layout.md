---
status: accepted
date: 2026-10-15
---

# 0014 Code layout: thin mains, an importable command tree, black-box tests under test/

**Status:** Accepted. The owner said yes at Gate 1 on 15 October 2026.

The evidence is in [the research note on Go project structure](../../research/go-project-structure.md), sections 1, 2.1 to 2.3 and 4.

## Context

The repo is about to go public, and the goal asks for a layout of `cmd/`, `internal/`, `examples/`, `scripts/`, `test/e2e/` and `testdata/`. Four things in today's tree stand in the way:

1. **The commands live in `package main`** in `cmd/cli`, which holds about 1,250 lines of cobra code. Nothing else can import them. So a generator cannot build the CLI reference and man pages from them, and tests cannot run the CLI in-process.
2. **Test tools sit in a top-level `e2e/` folder.** It mixes a Python suite with a Go tool (`ttyshot`) that the screenshots also use.
3. **`go install` names the client `cli`.** Go names an installed binary after its folder, and the name may not appear in paths, so `go install <module>/cmd/cli@latest` gives a command called `cli`. The goal asks for `go install` as an install path.
4. **Some files are long.** Eight hand-written files are over the 500-line bar, mostly in `internal/host`, `internal/client` and `internal/tui`.

The thirteen repos in the research show two shapes. Small tools keep `main.go` at the root. Projects with several binaries put each `main` in `cmd/<name>/`. Every application keeps its code in `internal/`. GitHub CLI and Crush keep the cobra tree in an importable package, and GitHub CLI's `cmd/gen-docs` builds its manual from it.

## Options

1. **Keep the layout.** Add a hidden `docs` command to the client, as Charm's tools add a hidden `man` command.
2. **One root `main.go` with `internal/cmd`**, as Crush does.
3. **Thin mains in `cmd/cli` and `cmd/host`, the command tree in `internal/cli`, dev tools as mains in `cmd/`, and black-box tests under `test/`.**
4. **Option 3, plus a public `pkg/`** for reusable parts such as `proto` and `ring`.
5. **Option 3, plus splitting `internal/host`, `internal/client` and `internal/tui` into sub-packages now.**
6. **Option 3, with the client's thin main at the root** instead of `cmd/cli`, so `go install <module>@latest` names the binary after the module.

## Decision

Option 6:

```text
main.go           the client's thin main: calls cli.Main (from cmd/cli)
cmd/host/         the host helper (unchanged)
cmd/docgen/       dev tool, never shipped: writes docs/reference/, man pages and completions
internal/cli/     the cobra command tree, moved from cmd/cli
internal/...      the other packages, unchanged
test/e2e/         the Python end-to-end suite (from e2e/)
test/ttyshot/     the pseudo-terminal renderer (from e2e/ttyshot)
testdata/         only fixtures shared by several packages, created with the first one
examples/, scripts/, vt/, install.sh   unchanged
```

The reasons:

- `go install <module>@latest` must give a binary with the product's name, and the name may only come from the module path. A main at the root is the one place that does this, as Crush, Gum, Glow, lazygit and k9s show. The host helper keeps `cmd/host`: it is never installed by hand.
- Option 2's `internal/cmd` would work too, but `internal/cli` says what it holds, and leaves `cmd/` for mains.
- An importable command tree is what GitHub CLI uses to generate its docs. It keeps doc code out of the shipped binary, which option 1 would not.
- We have no outside users of our packages, so a public `pkg/` (option 4) would only freeze an API.
- Splitting packages now (option 5) would be churn. The research mapped how the files use each other: `internal/client` and `internal/tui` are tightly knit. The one cheap later split is the shadow-terminal supervisor in `internal/host`, which depends only on `RingSize`. Long files are split within their packages, as a separate job.

`internal/brand` stays the only place that spells the product name. `cmd/docgen`, the site config and the scripts read it from there.

The move happens in the small steps in section 7 of the research note; step 1 puts the main at the root rather than in `cmd/cli`. Each step is one commit with behaviour unchanged. The `--help` output of every command, the golden tests and the end-to-end count must match the baseline.

## Consequences

- Good, because the CLI reference, man pages and completions can be generated from the code, as the goal asks.
- Good, because the layout matches what contributors know from GitHub CLI, restic and Soft Serve, and what the goal lists.
- Good, because test tooling has one clear home, apart from the product code.
- Bad, because moving `cmd/cli` to `internal/cli` touches every line of blame for those files. `git log --follow` still finds the history.
- Bad, because the build paths in the justfile, GoReleaser and CI change from `./cmd/cli` to `.`, in the same commit as the move.
- Bad, because the scripts that name paths must change in the same commits: `check-name.sh`, `screenshots.sh` and the `just e2e` recipe. Otherwise the name check could silently skip the moved Python files.
