# ADR: Git Bare Repository Unification

## Context
Currently, `CloneForDeploy` correctly uses a Git bare repository + worktree approach to optimize deployment speed and disk usage. However, `FetchAndGetCommits` uses a separate traditional `git clone` into `<projectName>` directory. This means each project has two copies of the repository downloaded, taking up double the disk space and taking double the network time to initialize.

## Decision
We will unify the Git management in `internal/infrastructure/git/client.go`. `FetchAndGetCommits` will be refactored to use the same `<projectName>.git` bare repository that `CloneForDeploy` uses. Since `git log` can be executed directly inside a bare repository (without needing a work tree), this is highly efficient.

## Consequences
- **Positive**: Reduces disk space by 50% for Git storage. Reduces initial clone time for a new project since both deployment and commit fetching will share the same bare repo.
- **Negative**: Need to handle the edge case where the bare repo is created by `FetchAndGetCommits` before the first deployment happens, meaning both methods must correctly initialize the bare repo if it does not exist.
