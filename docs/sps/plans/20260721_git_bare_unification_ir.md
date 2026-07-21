# Git Bare Repository Unification Implementation Plan

## Goal
Unify the Git management in `internal/infrastructure/git/client.go` to use a single bare repository (`<projectName>.git`) for both `CloneForDeploy` and `FetchAndGetCommits`, reducing disk footprint and network clone times by 50%.

## Proposed Changes

### 1. Extract Bare Repo Initialization
Extract the bare repo initialization logic (currently in `CloneForDeploy`) into a reusable method.

#### [MODIFY] internal/infrastructure/git/client.go
- Create a new method `ensureBareRepo(repoURL, bareRepoPath string, logChan chan<- string) error`
- Move lines 43-57 (the `os.Stat` and `git clone --bare` or `git fetch`) into this new method.
- Update `CloneForDeploy` to call `ensureBareRepo`.

### 2. Refactor FetchAndGetCommits
Refactor `FetchAndGetCommits` to use `ensureBareRepo` and run `git log` directly against the bare repository instead of doing a full clone.

#### [MODIFY] internal/infrastructure/git/client.go
- Change `workspacePath := filepath.Join(c.workspaceBase, projectName)` to `bareRepoPath := filepath.Join(c.workspaceBase, projectName+".git")`.
- Call `ensureBareRepo(repoURL, bareRepoPath, nil)` (we can pass a dummy or nil `logChan` if one isn't provided, though `ensureBareRepo` must handle a nil `logChan` safely).
- Update the `args` array for `git log` to run directly on the bare repository. We can use `--git-dir=bareRepoPath` or simply `cmd.Dir = bareRepoPath`.
- Change `origin/%s` to `%s` since in a bare repo, the branches fetched are local to the bare repo. Wait! If it's a bare clone, `--bare` maps `refs/heads/*` to `refs/heads/*`, or `refs/remotes/origin/*` depending on config. We used `+refs/heads/*:refs/heads/*` for fetching, so the branches are just local branches like `main`.
- Therefore, the `git log` arguments will be:
  - `git log <fromCommit>..<branch>` instead of `origin/<branch>`

### 3. Update Tests
Fix unit tests that mock or expect the old normal clone behavior.

#### [MODIFY] internal/infrastructure/git/client_test.go
- Ensure `TestClient_CloneOrPullWorktree` passes.
- Ensure `TestClient_FetchAndGetCommits` passes.

## Verification Plan
1. Run `go test -v ./internal/infrastructure/git`.
2. Observe disk space for `.git` and workspaces.
3. Ensure CI `golangci-lint` remains green.
