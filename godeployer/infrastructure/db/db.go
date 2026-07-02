// ============================================================
// 文件：db.go
// 作用：💾 数据库初始化和全局管理！
//
// 这个文件负责"数据库"的所有启动工作：
// 1. 连接数据库（支持 SQLite/MySQL/PostgreSQL）
// 2. 自动建表（AutoMigrate——根据结构体自动创建数据表）
// 3. 创建默认管理员（第一次运行时创建 admin 账号）
// 4. 修复卡住的任务（程序崩溃后，自动把"部署中"的任务标记为已取消）
//
// 默认使用 SQLite——一个"文件即数据库"的轻量级数据库。
// 不需要安装任何数据库软件，一个文件就能存全部数据！
//
// 给初二小白的比喻：
// 数据库就像 Excel 表格：
// - users 表 = 一个叫"用户"的工作表
// - deploy_tasks 表 = 一个叫"部署任务"的工作表
// - AutoMigrate = 自动创建工作表 + 画好表头
// - 一行记录 = 一个用户 / 一条部署任务
// ============================================================

package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"    // 💎 纯 Go 实现的 SQLite 驱动（不需要 C 编译器）
	"gorm.io/driver/mysql"           // 🐬 MySQL 驱动
	"gorm.io/driver/postgres"        // 🐘 PostgreSQL 驱动
	"gorm.io/gorm"                   // 🏗️ GORM：Go 的"超级 ORM"（对象关系映射）
	"golang.org/x/crypto/bcrypt"     // 🔐 bcrypt 加密

	"deploy/godeployer/domain"
)

// DB 全局数据库实例
var DB *gorm.DB

// ============================================================
// 🚀 InitGORM：初始化 GORM 数据库实例
//
// GORM 是什么？
// ORM = Object Relational Mapping（对象关系映射）
// 简单说：让你用操作"对象"的方式操作"数据库"。
// 不用写 SQL 语句，直接 .Create()、.Find()、.Update() 就行！
//
// 支持的数据库：
// - sqlite：默认，文件即数据库，零配置
// - mysql：流行的关系数据库
// - postgres：功能强大的关系数据库
// ============================================================

// InitGORM initializes GORM database instance for given driver and DSN
func InitGORM(driverName, dsn string) (*gorm.DB, error) {
	// 根据数据库类型选择不同的"拨号器"
	var dialector gorm.Dialector

	switch driverName {
	case "mysql":
		dialector = mysql.Open(dsn)     // 🐬 MySQL 连接
	case "postgres":
		dialector = postgres.Open(dsn)  // 🐘 PostgreSQL 连接
	case "sqlite":
		fallthrough                     // 默认走 SQLite
	default:
		dialector = sqlite.Open(dsn)    // 💎 SQLite 连接
	}

	// 打开数据库连接
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// SQLite 特殊设置：只允许一个写连接
	// 因为 SQLite 同时只能有一个写入操作，多了会报"数据库被锁定"
	if driverName == "sqlite" {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.SetMaxOpenConns(1) // 🚦 最多 1 个连接
		}
	}

	// 自动建表（如果表不存在就创建，有变化就更新）
	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	// 创建默认管理员账号（第一次运行时）
	if err := createDefaultAdmin(db); err != nil {
		return nil, fmt.Errorf("failed to seed admin: %w", err)
	}

	// 修复卡住的任务（程序崩溃后清理）
	if err := repairStalledTasks(db); err != nil {
		return nil, fmt.Errorf("failed to auto-repair stalled tasks: %w", err)
	}

	DB = db // 存入全局变量
	return db, nil
}

// ============================================================
// 🏗️ autoMigrate：自动迁移建表
//
// GORM 的 AutoMigrate 会根据结构体定义自动创建/更新数据库表。
// 比如 domain.UserResponse 有 Username、Role 等字段，
// AutoMigrate 就会在数据库中创建 users 表，列名就是这些字段。
//
// 如果表已经存在且结构没变，什么也不做。
// 如果加了新字段，自动添加新列（不会删除已有列）。
// ============================================================

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.UserResponse{}, // 👤 users 表
		&domain.DeployTask{},   // 📋 deploy_tasks 表
	)
}

// ============================================================
// 👤 createDefaultAdmin：创建默认管理员
//
// 第一次启动系统时，如果数据库里一个用户都没有，
// 自动创建一个 admin 账号（默认密码 admin123）。
//
// 也可以设置环境变量 ADMIN_PASSWORD 来指定密码。
// 每次启动都会确保 admin 用户的角色是"admin"。
// ============================================================

func createDefaultAdmin(db *gorm.DB) error {
	var count int64
	// 统计有多少用户
	if err := db.Model(&domain.UserResponse{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		// 没有任何用户？创建默认管理员！
		pwd := os.Getenv("ADMIN_PASSWORD") // 试试环境变量
		if pwd == "" {
			pwd = "admin123" // 默认密码
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		admin := &domain.UserResponse{
			Username:     "admin",
			PasswordHash: string(hash),
			Role:         "admin",
			CreatedAt:    time.Now(),
		}
		if err := db.Create(admin).Error; err != nil {
			return err
		}
	} else {
		// 确保 admin 用户始终有管理员角色
		db.Model(&domain.UserResponse{}).
			Where("username = ?", "admin").
			Update("role", "admin")
	}
	return nil
}

// ============================================================
// 🔧 repairStalledTasks：修复卡住的任务
//
// 如果系统崩溃了，数据库里可能还有状态是"部署中"或"等待中"的任务。
// 这些任务永远不会完成，永远卡在那里……
// 这个函数把它们全部标记为"已取消"（aborted）。
// 就像：游戏打到一半闪退了，重新打开后之前的进度作废~
// ============================================================

func repairStalledTasks(db *gorm.DB) error {
	return db.Model(&domain.DeployTask{}).
		Where("status IN ?", []domain.DeployStatus{
			domain.StatusPending,      // ⏳ 等待中的
			domain.StatusDeploying,    // 🚀 部署中的
		}).
		Update("status", string(domain.StatusAborted)). // → 变成已取消
		Error
}

// ============================================================
// 🧪 InitTestDB：测试用的初始化函数
//
// 为了方便写测试，提供一个快捷方式：
// 初始化数据库、创建任务仓库，一次性搞定！
// ============================================================

// InitTestDB provides a unified helper for tests to get both sql.DB and TaskRepository
func InitTestDB(dsn string) (*sql.DB, domain.TaskRepository, error) {
	gormDB, err := InitGORM("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, err
	}
	taskRepo := NewTaskRepository(gormDB)
	return sqlDB, taskRepo, nil
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 什么是 ORM？
//    A: 让你用操作对象的方式操作数据库。不用写 SQL 了！
//       比如 db.Create(&user) 就相当于 INSERT INTO users ...
//
// 2. Q: SQLite 的优缺点？
//    A: 优点：零配置、一个文件、适合小项目
//       缺点：并发写入差、不支持高并发、功能较少~
//
// 中级：
// 3. Q: AutoMigrate 和直接执行 SQL 建表的区别？
//    A: AutoMigrate 自动根据 Go 结构体建表，
//       改代码后自动加列，但不会删列（安全考虑）。
//       直接 SQL 更灵活但需要手动维护~
//
// 4. Q: repairStalledTasks 为什么要修复"卡住"的任务？
//    A: 如果程序部署到一半崩溃了，重启后那些"部署中"的任务
//       无法自动完成。修复它们标记为取消，让队列可以继续接受新任务~
//
// 高级：
// 5. Q: 为什么 SetMaxOpenConns(1) 只对 SQLite 设置？
//    A: SQLite 不支持并发写入！同时多个写入会导致 "database is locked" 错误。
//       MySQL/PostgreSQL 有多写能力，不需要这个限制~
// ============================================================
