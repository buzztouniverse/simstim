---
status: accepted
date: 2026-10-15
---

# 0010 No sign-off or attribution lines in commits

This record replaces the DCO part of [0005](0005-apache-2-and-dco.md). The licence stays Apache-2.0.

## Context

0005 asked every commit to carry a `Signed-off-by:` line (the Developer Certificate of Origin). Agent sessions also added a `Claude-Session:` link to some commits. The owner wants commit messages to hold only the change and its reason: no trailers of any kind.

## Options

1. Keep the DCO sign-off for every commit.
2. No sign-off or attribution lines. Contributions are covered by the licence itself: section 5 of Apache-2.0 says that a contribution submitted to the project is under the same licence, unless the contributor says otherwise.

## Decision

Option 2. Commit messages are a subject, a blank line and a body. They have no `Signed-off-by:`, `Co-Authored-By:` or `Claude-Session:` lines, and no session links or "Generated with" lines. The history was rewritten on 15 October 2026 to remove the lines that earlier commits carried.

## Consequences

- Good, because the history reads cleanly, and agents have one simple rule.
- Bad, because there is no per-commit certificate of origin. Apache-2.0 section 5 covers contributions. A contributor licence agreement or the DCO can come back with a new record if the project needs one.
