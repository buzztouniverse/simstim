---
status: accepted
date: 2026-10-15
---

# 0016 Public repo layout: full community files at the root, GitHub config in .github/, private material out

**Status:** Accepted. The owner said yes at Gate 1 on 15 October 2026.

The evidence is in [the research note on Go project structure](../research/go-project-structure.md), sections 2.4, 2.5, 2.8 and 6.

## Context

The repo is private today and must be ready to go public at any moment. Several things are already done:

- the goal, status, test-host log and owner checklist have moved to the git-ignored `private/` folder;
- `just check-private` guards against personal paths, private addresses and emails;
- there is a licence, NOTICE, CHANGELOG, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, a PR template, two issue forms, Dependabot and CI.

The goal asks for more:

- SUPPORT, GOVERNANCE, MAINTAINERS and a short AGENTS.md;
- issue forms for providers and docs, labels as code, Discussions and release notes;
- an OpenSSF Scorecard of 8 or more;
- signed releases with an SBOM and SLSA provenance.

The thirteen repos in the research differ a lot. Charm keeps its community files in an org-wide `.github` repository. GitHub CLI keeps them in `.github/`. restic adds GOVERNANCE.md. None of them has SUPPORT or MAINTAINERS in the repo itself, and none keeps its labels as code. GoReleaser has the highest Scorecard (8.7): it pins every action by SHA, sets permissions per job, and runs CodeQL and Scorecard.

## Options

Where the community files live:

1. Keep only README and LICENSE in the repo, and inherit the rest from an organisation's `.github` repository (Charm). We have no organisation.
2. Put the community files in `.github/` (GitHub CLI).
3. **Put the full set at the root, and keep `.github/` for GitHub's own config.**

Where release notes come from:

1. GoReleaser groups by Conventional Commit prefix (Charm, GoReleaser).
2. GitHub's native notes, grouped by label (lazygit).
3. release-drafter, from pull request titles.
4. **The version's section of CHANGELOG.md, passed to GoReleaser with `--release-notes`.**

Labels:

1. Made by hand in the settings.
2. **Kept as code in `.github/labels.yml`, synced by a workflow.**

## Decision

The chosen options are root files, CHANGELOG notes and labels as code.

**Root:** README, LICENSE, NOTICE, CHANGELOG, CONTRIBUTING, CODE_OF_CONDUCT, SECURITY (add how to verify a release), SUPPORT, GOVERNANCE (maintainer-led, through decision records), MAINTAINERS, and a short AGENTS.md that points to CONTRIBUTING and `docs/contributing/`. Add `.gitattributes`, which marks generated docs and golden files as generated. The docs-check configs are dotfiles.

**`.github/`:**

- CODEOWNERS (GitHub reads it here first);
- `dependabot.yml` for Go (the root and `site/`) and for actions;
- `labels.yml`;
- issue forms: bug, feature, provider and docs, with `config.yml` turning blank issues off and linking to private security reports, Discussions and the docs;
- Discussion forms for Ideas and Q&A;
- the PR template;
- workflows: `ci.yml`, `docs.yml`, `release.yml`, `codeql.yml`, `scorecard.yml` and `labels.yml`.

Every action is pinned to a full SHA with a version comment, and permissions are set per job. zizmor checks the workflows.

**Labels:**

- kinds: `bug`, `enhancement`, `docs`;
- areas: `area: client`, `area: host`, `area: launcher`, `area: attach`, `area: providers`, `area: docs`, `area: ci`;
- triage: `needs triage`, `needs design`, `help wanted`, `good first issue`;
- roadmap: `now`, `next`, `later`.

`crazy-max/ghaction-github-labeler` syncs them, as a dry run on pull requests.

**Releases:**

- notes from the CHANGELOG section for the tag;
- a cosign signature bundle on `checksums.txt`;
- an SBOM per archive;
- GitHub artifact attestations on the checksums;
- the docs site for that version as an asset;
- a Homebrew cask, rather than a formula, which GoReleaser has deprecated.

Scorecard gives full marks for signed releases only when a `*.intoto.jsonl` provenance file is a release asset. Whether to publish one is tested at the first release.

**Private material** stays out of the tree: `private/` and `docs/research/private/` are git-ignored, and `just check-private` runs in CI.

**Settings the owner sets when going public** (they are not code): Discussions categories (Announcements, Ideas, Q&A, Show and tell), private vulnerability reporting, branch protection, and Pages. They are listed in `docs/contributing/going-public.md`.

The reasons:

- With no organisation to inherit from, the root is where people and GitHub look first.
- The CHANGELOG is already written with each change, so it is the one source for notes. release-drafter would repeat it from pull request titles.
- Labels as code make the triage and good-first-issue scheme reviewable and repeatable.

## Consequences

- Good, because a newcomer finds every community file at the root, and GitHub's community profile is complete.
- Good, because pinned actions, per-job permissions, CodeQL, Scorecard, signed releases and SBOMs cover most Scorecard checks.
- Good, because release notes and the CHANGELOG cannot disagree.
- Bad, because the root grows by four files and four dotfiles.
- Bad, because the CHANGELOG discipline must hold. A change without an entry has no release note, so the docs-impact check in CI must also look for a CHANGELOG entry.
- Bad, because a single maintainer who merges their own changes is unlikely to score well on Scorecard's Code-Review check, and some checks (branch protection, the Best Practices badge) depend on settings only the owner can change.
- Bad, because Dependabot will open more pull requests once every action is pinned by SHA.
