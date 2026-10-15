# Keep each task in its own worktree

A session can get its own git worktree and branch, so several agents can work in one repository without touching each other's files.

```sh
simstim new my-server --agent claude --dir ~/src/app --name auth-fix --worktree --detach
```

The session starts in a new worktree next to the repository, on a branch named after the session. `--base main` starts the branch from `main` instead of the current `HEAD`.

- The worktree is locked while the session lives, so `git worktree prune` leaves it alone.
- Files listed in the repository's `.worktreeinclude` (one pattern a line, for example `.env`) are copied into the new worktree.
- If the repository has a [setup script](setup-hooks.md), it runs in the new worktree after the copy, before the agent, once you trust the repository.
- When you delete the session, the worktree is removed only if nothing would be lost: no uncommitted changes and no commits that are not on another branch. Otherwise it stays, and the client says where.

In the launcher, press `n`, then `+wt` at the end of the quick-start line, or `ctrl+f` for the form, which has a worktree switch.
