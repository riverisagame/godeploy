---
name: godeployer
description: "Skill for the Godeployer area of godeploy. 133 symbols across 29 files."
---

# Godeployer

133 symbols | 29 files | Cohesion: 69%

## When to Use

- Working with code in `godeployer/`
- Understanding how TestAPI_SystemPrune_Permissions, SetupTestRouter, TestAPI_LoginVerify work
- Modifying godeployer-related functionality

## Key Files

| File | Symbols |
|------|---------|
| `godeployer/api.go` | SetupRoutes, HandleGetProjectCommits, HandleGetProjectPreviewDiff, HandleCreateTask, isCommitHash (+16) |
| `godeployer/engine.go` | NewDeployEngine, RunLocalBuild, RunDeploy, runCmd, StartDispatcher (+9) |
| `godeployer/api_test.go` | SetupTestRouter, TestAPI_LoginVerify, TestAPI_GetProjectsVerify, TestAPI_CreateTaskAudit, TestAPI_DeployLockVerify (+8) |
| `godeployer/engine_test.go` | Close, TestEngine_LocalBuildVerify, TestEngine_DeployTimeout, TestEngine_MultiNodeDeploy, TestDeployEngine_ExcludeInjection (+6) |
| `godeployer/auth.go` | GenerateToken, ParseToken, HashPassword, CheckPasswordHash, AuthMiddleware (+1) |
| `godeployer/git.go` | getCacheDir, EnsureRepoCache, GetCommits, GetDiff, GetDiffForFile (+1) |
| `godeployer/api_enhance_test.go` | TestAPI_JSON_ChangesCache, TestAPI_DualDiff_PersistenceAndFallback, SetupEnhanceTestRouter, TestAPI_CreateTaskWithDescription, TestAPI_PreviewDiffWithFileList |
| `godeployer/db_test.go` | TestDB_InitVerify, TestDB_SeedDefaultAdmin, TestDB_StartupResilience, TestDB_Migration_Role, TestDB_ConcurrentTaskUpdates |
| `godeployer/notifier.go` | NewEventBus, Register, Publish, StartEventConsumer, Close |
| `godeployer/ssh_pool.go` | NewSSHPool, createClient, Get, Put, Close |

## Entry Points

Start here when exploring this area:

- **`TestAPI_SystemPrune_Permissions`** (Function) — `godeployer/api_prune_test.go:14`
- **`SetupTestRouter`** (Function) — `godeployer/api_test.go:22`
- **`TestAPI_LoginVerify`** (Function) — `godeployer/api_test.go:130`
- **`TestAPI_GetProjectsVerify`** (Function) — `godeployer/api_test.go:171`
- **`TestAPI_CreateTaskAudit`** (Function) — `godeployer/api_test.go:199`

## Key Symbols

| Symbol | Type | File | Line |
|--------|------|------|------|
| `TestAPI_SystemPrune_Permissions` | Function | `godeployer/api_prune_test.go` | 14 |
| `SetupTestRouter` | Function | `godeployer/api_test.go` | 22 |
| `TestAPI_LoginVerify` | Function | `godeployer/api_test.go` | 130 |
| `TestAPI_GetProjectsVerify` | Function | `godeployer/api_test.go` | 171 |
| `TestAPI_CreateTaskAudit` | Function | `godeployer/api_test.go` | 199 |
| `TestAPI_DeployLockVerify` | Function | `godeployer/api_test.go` | 234 |
| `TestAPI_GetTaskLogTruncate` | Function | `godeployer/api_test.go` | 268 |
| `TestAPI_GitDiffVerify` | Function | `godeployer/api_test.go` | 318 |
| `TestAPI_GetTaskDiff_RaceCondition` | Function | `godeployer/api_test.go` | 380 |
| `TestHandleTasks_InvalidProject` | Function | `godeployer/api_test.go` | 447 |
| `TestHandleDeploy_InvalidEnv` | Function | `godeployer/api_test.go` | 473 |
| `TestAPI_UserManagement_CRUD` | Function | `godeployer/api_user_test.go` | 10 |
| `TestAPI_WS_TaskLog_Integration` | Function | `godeployer/api_ws_test.go` | 13 |
| `GenerateToken` | Function | `godeployer/auth.go` | 33 |
| `SetupRoutes` | Function | `godeployer/api.go` | 34 |
| `TestAPI_JSON_ChangesCache` | Function | `godeployer/api_enhance_test.go` | 166 |
| `TestAPI_DualDiff_PersistenceAndFallback` | Function | `godeployer/api_enhance_test.go` | 260 |
| `TestProjectPermissions` | Function | `godeployer/api_permissions_test.go` | 11 |
| `TestAPI_SystemPrune_OrphanCleanup` | Function | `godeployer/api_prune_test.go` | 38 |
| `TestAPI_DiffCache_MaxSizeLimit` | Function | `godeployer/api_prune_test.go` | 134 |

## Execution Flows

| Flow | Type | Steps |
|------|------|-------|
| `Main → Claims` | cross_community | 7 |
| `RunRollback → SSHPool` | cross_community | 6 |
| `Main → CreateDefaultAdmin` | cross_community | 5 |
| `Main → RepairStalledTasks` | cross_community | 5 |
| `Main → APIHandler` | cross_community | 5 |
| `Main → RoleMiddleware` | cross_community | 5 |
| `RunRollback → SSHExecutor` | cross_community | 5 |
| `Main → LoadConfig` | cross_community | 4 |
| `Main → EventBus` | cross_community | 4 |
| `Main → DeployEngine` | intra_community | 4 |

## How to Explore

1. `gitnexus_context({name: "TestAPI_SystemPrune_Permissions"})` — see callers and callees
2. `gitnexus_query({query: "godeployer"})` — find related execution flows
3. Read key files listed above for implementation details
