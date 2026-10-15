# How the docs work

The docs live in `docs/`, as plain Markdown that reads well on GitHub. The same files build the docs site.

## Where a page goes

The user docs follow [Diátaxis](https://diataxis.fr/): each page does one of four jobs.

| Folder | Job | Example |
| --- | --- | --- |
| `docs/tutorials/` | Teach by doing, start to finish | The first five minutes |
| `docs/how-to/` | Do one task | Add a host; get alerts |
| `docs/reference/` | Look something up. **Generated; never edit by hand.** | Every command and flag |
| `docs/explanation/` | Understand why | The fidelity contract |

Contributor docs are in `docs/contributing/`, the evidence behind the design in `docs/research/`, and what comes next in `docs/roadmap.md`.

## Docs ship with the change

A change that a user can notice updates its docs and adds a line to `CHANGELOG.md` under Unreleased, in the same commit. CI fails a pull request that changes commands, config, keys or agents without touching the docs and the CHANGELOG, unless it is labelled `no docs needed`.

## The generated reference

`just docs` runs `cmd/docgen`, which writes `docs/reference/` from the code:

- a page per command, from the command tree;
- `config.toml` and `hosts.toml`, from the comments on the settings structs in `internal/client`;
- the launcher's keys, from its command registry;
- the agent table, from `internal/agents`.

To change a reference page, change the code or its comment, then run `just docs`. CI fails when the committed pages differ from a fresh run. The release build also writes man pages and shell completions this way.

## The site

`just docs-site` builds the site into `build/site/public` with Hugo and the Hextra theme (`site/`). `cmd/docgen -site` first turns the Markdown into the site's content: each `README.md` becomes its section's page, and each page's first heading its title. The community files at the root of the repository (`CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md` and the rest) become the Project section, at `/project/`.

`cmd/docgen` records each page's path in the repository, and writes `data/pages.json`, which maps every hosted file to its page. So a link between two Markdown files goes to that page on the site, never to GitHub, and works on GitHub too. A link to any other file, such as code or a script, goes to its source on GitHub. `just docs-check` fails if a built page links to GitHub for a page the site hosts (`scripts/check-site-links.py`).

CI builds the site on every push and keeps it as an artifact named `docs-site`. Once the repository is public, GitHub Pages serves the latest release at the root, each release at `/vX.Y/`, and `main` at `/dev/`.

## Checks

`just docs-check` runs what CI runs:

- the generated reference is current;
- Markdown lint (markdownlint-cli2; `.markdownlint-cli2.yaml`);
- British spelling (typos; `.typos.toml`);
- plain English (Vale, with the project's small style in `.vale/styles/Plain`);
- every command in the tutorials and how-to guides: each must exist with its flags, and with `DOCS_HOST` set, `just test-docs` runs them on a test host (a comment before a block can mark it `skip` or `help`);
- every link on the built site (lychee, offline). `just docs-check --external` also checks links to other sites, which CI does on `main` and weekly;
- no built page links to GitHub for a page the site hosts.

## Images

Screenshots and recordings come from scripts, never by hand. `just screenshots` runs the `demo` command, where nothing connects anywhere, inside `test/ttyshot`:

- the stills in `docs/images/*.svg` come from the shot list in `scripts/screenshots.sh`;
- the recorded demos come from the tapes in `docs/demos/*.tape`, with motion on. Each tape is a script of keys and pauses (`send`, `type`, `sleep`, `wait`) between `record start` and `record save`, which writes an animated SVG (it stands still for readers who ask for reduced motion) and an asciicast (`.cast`) for asciinema's player.

Run it after any change to the launcher, so the images always show the current build.
