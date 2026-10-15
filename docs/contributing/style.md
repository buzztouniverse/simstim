# Style

## Code

- **The product name lives in `internal/brand`.** Use `brand.Name`, `brand.HostBinary` and `brand.Env(...)`; never spell the name in code, paths, environment variables or unit names. `just check-name` checks.
- **Byte-exact while attached.** Nothing may change the bytes between an agent's terminal and the user's, except the two keys the input filter owns. A change near the attach path needs a test that proves the bytes are unchanged.
- **Agents are data**, in `internal/agents`. No other code branches on an agent's name.
- **Errors say what to try next.** A user-facing error is a `client.UserError` with a message and a hint. Wrap other errors with context: `fmt.Errorf("read %s: %w", path, err)`.
- **Every exported name has a doc comment.**
- **Small files and functions:** a file over 500 lines, or a function that is hard to follow, is split.
- **No new listening ports, relays or telemetry.**
- `just lint` runs `go vet`, golangci-lint (`.golangci.yml`), and the name, private-detail and script checks.

## Writing

The docs, the CLI's messages and the commit messages share one style:

- **Plain English, short sentences.** One idea per sentence. Prefer the short, common word over the long one.
- **British spelling:** colour, behaviour, licence (noun).
- **Say what to do,** in the active voice: "Run `just docs`", not "`just docs` should be run".
- **Name things as users see them.** A setting is a key in `config.toml`; a screen is what the user sees.
- **Be exact.** Numbers with units, versions, and links to sources.
- `just docs-check` checks spelling and some of the wording.

## Commits

One logical change per commit. The message is a subject (the change, in the imperative, under about 70 characters), a blank line, and a body that says why. No trailers of any kind, and never `git commit -s` ([0010](decisions/0010-no-commit-trailers.md)).
