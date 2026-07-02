// ============================================================
// 文件：user_repository.go
// 作用：👤 用户仓库——实现 domain.UserRepository 接口！
//
// 这个文件负责用户（UserResponse）的所有数据库操作：
// - 按用户名查询（GetUserByUsername）
// - 创建用户（CreateUser）
// - 更新用户（UpdateUser）
// - 获取所有用户（GetUsers）
// - 删除用户（DeleteUser）
// ============================================================

package db

import (
	"deploy/godeployer/domain" // 📋 领域接口和实体
	"gorm.io/gorm"             // 🏗️ GORM
)

// userRepository 包内私有的结构体——实现了 domain.UserRepository
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户资源库
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// GetUserByUsername 根据用户名查找用户（登录时用的）
func (r *userRepository) GetUserByUsername(username string) (*domain.UserResponse, error) {
	var user domain.UserResponse
	// SELECT * FROM users WHERE username = ? LIMIT 1
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 没找到（不是错误）
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建新用户
// passwordHash 是 bcrypt 加密后的密码哈希值
func (r *userRepository) CreateUser(user *domain.UserResponse, passwordHash string) error {
	user.PasswordHash = passwordHash // 存入密码哈希
	return r.db.Create(user).Error   // INSERT INTO users ...
}

// UpdateUser 更新用户信息（角色、权限、密码等）
// 只更新非零字段：role、bound_git_authors、permitted_projects
// 如果传了 passwordHash，也更新密码
func (r *userRepository) UpdateUser(user *domain.UserResponse, passwordHash string) error {
	// 用 map 存要更新的字段（只更新需要改的）
	updates := map[string]interface{}{
		"role":                 user.Role,
		"bound_git_authors":    user.BoundGitAuthors,
		"restrict_git_authors": user.RestrictGitAuthors,
		"permitted_projects":   user.PermittedProjects,
	}
	// 如果传了新密码，也更新
	if passwordHash != "" {
		updates["password_hash"] = passwordHash
	}
	return r.db.Model(user).Updates(updates).Error
}

// GetUsers 获取所有用户的列表
func (r *userRepository) GetUsers() ([]domain.UserResponse, error) {
	var users []domain.UserResponse
	// SELECT * FROM users
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteUser 根据用户名删除用户
func (r *userRepository) DeleteUser(username string) error {
	return r.db.Where("username = ?", username).Delete(&domain.UserResponse{}).Error
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 用户仓库和任务仓库为什么分开写？
//    A: 单一职责原则！每个仓库只管一种数据的操作，
//       用户仓库管用户，任务仓库管任务，互不干扰~
//
// 中级：
// 2. Q: UpdateUser 为什么用 map 更新而不是直接 .Save()？
//    A: .Save() 会更新所有字段（包括零值），
//       用 map 可以只更新指定的字段，防止意外把其他字段清空~
//
// 高级：
// 3. Q: 如果以后换数据库（从 SQLite 换到 MySQL），需要改哪些代码？
//    A: 只需要改 db.go 里的驱动初始化！
//       这些 repository 文件完全不用改，因为 GORM 统一了不同数据库的操作~
// ============================================================
