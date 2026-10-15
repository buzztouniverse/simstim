# Going public

The repository is private until its owner makes it public. This page lists what is already done, what `just public-check` proves, and the owner's steps on the day.

## Already done

- Private material lives only in git-ignored folders: `private/` and `docs/research/private/`. `just check-private` fails on personal paths, private addresses, tailnet names and emails, and on the owner's own words in `private/deny.txt`.
- The history was audited with gitleaks (no secrets), and rewritten once, with the owner's yes, to remove private details from old commits. CI scans the whole history on every push.
- Every action is pinned by SHA, tokens are least-privilege, and zizmor lints the workflows.
- Releases are signed (SSH key, cosign) with SBOMs and SLSA provenance, and the builds are reproducible.
- Community files are at the root and in `.github/`.

## Before you flip the switch

Run `just public-check`. It must pass. Then read the README and the docs home once more as a stranger would.

## The owner's steps

1. **Make the repository public:** Settings → General → Danger zone → Change visibility.
2. **Turn on private vulnerability reporting:** Settings → Security → Private vulnerability reporting. SECURITY.md links to it.
3. **Turn on Discussions:** Settings → General → Features → Discussions. The forms in `.github/DISCUSSION_TEMPLATE/` appear by themselves.
4. **Protect `main`:** Settings → Rules → New branch ruleset: block force pushes and deletion, and require the CI checks to pass.
5. **Turn on GitHub Pages:** Settings → Pages → Source: GitHub Actions. The docs workflow then deploys the site.
6. **Publish the Homebrew tap:** create the repository `homebrew-tap`, add a fine-grained token that can write to it as the Actions secret `HOMEBREW_TAP_TOKEN`, and the next release pushes the cask.
7. **Cut the first release** with the steps in [releasing](releasing.md).
8. **Check the Scorecard:** the Scorecard workflow runs on the next push to `main`. Add its badge to the README when the score is 8 or more. Apply for the OpenSSF Best Practices badge at <https://www.bestpractices.dev>, which also raises the score.
9. **Ask GitHub Support to purge cached views** of the commits from before the history rewrite, if any were ever shared.
10. **Announce it.**
