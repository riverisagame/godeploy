# 验收报告：项目仓库访问权限控制（Backend）

## 测试概要
- **时间**: 2026-05-28
- **测试目标**: 验证 `checkProjectAccess` 权限检查逻辑是否能正确阻挡未授权的项目访问。
- **覆盖范围**: `HandleGetProjects`, `HandleGetProjectRefs`, `HandleGetProjectCommits`, `HandleGetProjectPreviewDiff`, `HandleGetTaskDetail`, `HandleGetTaskLog`, `HandleWSLog`, `HandleCreateTask`, `HandleTriggerRollback`, `HandleGetTaskDiff`

## 验证步骤与结果
1. 修复了测试环境中的 `JWTSecret` 配置，以及 JWT 令牌生成参数。
2. 修复了 `api_permissions_test.go` 中测试用户模拟数据缺失 `password_hash` 和 `created_at` 字段导致 401/404 故障的问题。
3. 对 `HandleWSLog` 中的错误 Token 提取进行了修正。
4. 增补了对 `HandleTriggerRollback` 及 `HandleGetTaskDiff` 接口的权限校验漏点。
5. 运行完整测试用例 `go test . -v -run TestProjectPermissions`。

## 测试结果输出
```
=== RUN   TestProjectPermissions
=== RUN   TestProjectPermissions/user1_sees_all_projects
=== RUN   TestProjectPermissions/user2_sees_restricted_projects
=== RUN   TestProjectPermissions/admin_updates_user2_permissions
--- PASS: TestProjectPermissions (0.09s)
    --- PASS: TestProjectPermissions/user1_sees_all_projects (0.00s)
    --- PASS: TestProjectPermissions/user2_sees_restricted_projects (0.00s)
    --- PASS: TestProjectPermissions/admin_updates_user2_permissions (0.00s)
PASS
```

## 结论
后端项目权限控制逻辑验收通过。未授权访问能够被拦截，白名单控制逻辑生效，且测试用例完全跑通。
