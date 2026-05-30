# 多数据库支持与完全 DDD 化 (MySQL / PostgreSQL / SQLite)

## 背景描述
目前项目底层直接依赖 SQLite (infrastructure/sqlite) 并且在 application/deploy_service.go 和 interfaces/api/api.go 中存在大量直接调用 *sql.DB 的裸 SQL 查询。这导致代码无法无缝切换到其他数据库类型（如 MySQL、PostgreSQL），并且尚未达到真正的 DDD 标准结构。
为完成多数据库支持（SQLite、MySQL、PostgreSQL）且不影响现有功能，我们需要将所有裸 SQL 迁移到 domain/repository.go 定义的接口中，并在 infrastructure 层提供适配多方言的通用 DB 引擎。

## Proposed Changes
1. 在 domain 补充 DeployTask 实体与完整 Repo 接口。
2. 在 infrastructure/db 下重构数据库初始与方言绑定，引入 MySQL/Pg 驱动。
3. 将所有散落的 e.db.Query 替换为 domain.Repository 调用。
4. 确保 SQLite 仍完美兼容，并且无任何功能衰退。
