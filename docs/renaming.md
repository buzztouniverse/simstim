# Renaming the product

The name is a working name. The code keeps it in one place, so a rename is quick ([decision 0007](decisions/0007-name-in-one-place.md)).

## Where the name lives

| Place | How it is used |
| --- | --- |
| `internal/brand/brand.go` | The only spelling in code. Binaries, folders, environment variables, systemd units and the wire protocol all derive from it. |
| `go.mod` | The module path, which follows the GitHub repo name |
| The docs | Plain text |

The justfile and the release workflow read the name from `internal/brand/brand.go`. `just check-name` fails if the name is spelled anywhere else in code.

## How to rename

```sh
just rename newname                                     # name only
scripts/rename.sh newname github.com/iAmLakshya/newname # name and module path
```

The script changes the brand package, the module path and imports (if given), and the docs, then runs the name check. It prints the steps that need a person:

1. Rename the GitHub repo: `gh repo rename newname`.
2. Rename the local folder and update the git remote.
3. Hosts that already run the old helper keep their sessions under `~/.local/share/<old>` and `~/.local/state/<old>`. Remove them after those sessions end.
4. Update the Homebrew tap and the install script URLs.
