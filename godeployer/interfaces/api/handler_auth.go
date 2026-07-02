// ============================================================
// 文件：handler_auth.go
// 作用：🔑 登录 API——验证用户名密码，签发 JWT 令牌！
//
// 登录流程：
// 1. 用户发送用户名和密码
// 2. 从数据库查询对应用户的密码哈希
// 3. 用 bcrypt 比对密码是否正确
// 4. 正确 → 生成一个 JWT 令牌（有效期 24 小时）
// 5. 返回令牌给用户，以后每次请求都带这个令牌
//
// 给初二小白的比喻：
// 登录就像去游乐园🎡：
// 1. 在门口买票（输入用户名密码）
// 2. 工作人员验证（比对密码）
// 3. 给你一个手环（JWT 令牌）
// 4. 戴着手环随便玩（令牌有效期内不用再登录）
// ============================================================

package api

import (
	"net/http"
	"time"

	"deploy/godeployer/application"

	"github.com/gin-gonic/gin"
)

// LoginRequest 用户登录时发送的请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 📛 用户名（必填）
	Password string `json:"password" binding:"required"` // 🔑 密码（必填）
}

// HandleLogin 处理用户登录请求
func (h *APIHandler) HandleLogin(c *gin.Context) {
	// 解析请求体中的用户名和密码
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从数据库查询用户的密码哈希和角色
	var passwordHash string
	var role string
	err := h.db.QueryRow(
		"SELECT password_hash, role FROM users WHERE username = ?", req.Username,
	).Scan(&passwordHash, &role)
	if err != nil {
		// 用户名不存在！为了安全，不说"用户名不存在"或"密码错误"，
		// 统一说"用户名或密码错误"，防止攻击者试探有效用户名
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// 比对密码
	if !application.CheckPasswordHash(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	// ✅ 密码正确！签发 JWT 令牌
	// 令牌有效期 24 小时
	token, err := application.GenerateToken(req.Username, role, h.config.Global.JWTSecret, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// 返回令牌和用户信息
	c.JSON(http.StatusOK, gin.H{
		"token":    token,          // 🎫 JWT 令牌
		"username": req.Username,   // 👤 用户名
		"role":     role,           // 👑 角色
	})
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级：
// 1. Q: 为什么密码错误时要统一说"用户名或密码错误"？
//    A: 防止攻击者枚举有效用户名！
//       如果系统说"用户名不存在"，黑客就知道哪些用户名是注册过的。
//       统一回复让攻击者无法判断~
//
// 中级：
// 2. Q: JWT 的有效期为什么是 24 小时？
//    A: 安全与便利的平衡。太短（1 小时）用户要频繁登录，
//       太长（7 天）令牌泄露风险大。24 小时是常见做法~
// ============================================================
