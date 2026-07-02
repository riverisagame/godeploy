// ============================================================
// 文件：auth_service.go
// 作用：🔐 认证和授权服务——谁可以登录？可以做什么？
//
// 这个文件处理所有"安全相关"的事情：
// 1. 密码加密（HashPassword）：存密码不能明文存！
// 2. 密码验证（CheckPasswordHash）：登录时检查密码对不对
// 3. JWT 令牌（GenerateToken / ParseToken）：登录成功后发一个"通行证"
// 4. 鉴权中间件（AuthMiddleware）：检查请求有没有带有效的"通行证"
// 5. 角色权限（RoleMiddleware）：根据不同角色限制能做什么
//
// 给初二小白的比喻：
// 这就像游乐园的"安检+门票"系统：
// - HashPassword = 你的指纹信息（不可逆加密，不能还原成原始指纹）
// - JWT Token = 游乐园手环（上面写着你的身份和有效期）
// - AuthMiddleware = 门口检票（没手环？不让进！）
// - RoleMiddleware = 不同项目有不同权限（普通票不能玩过山车）
// ============================================================

package application

import (
	"errors"  // ❌ 错误处理
	"fmt"     // ✏️ 格式化字符串
	"net/http" // 🌐 HTTP 状态码
	"strings" // 📏 字符串处理（分割 Bearer token）
	"time"    // ⏰ 时间（设置 token 过期）

	"github.com/gin-gonic/gin"          // 🚄 Gin Web 框架
	"github.com/golang-jwt/jwt/v5"      // 🔑 JWT 库：签发和验证令牌
	"golang.org/x/crypto/bcrypt"         // 🔒 bcrypt 加密库：安全地存密码
)

// ============================================================
// 🔐 bcrypt 密码加密
//
// 为什么不能明文存密码？
// 如果数据库被黑了，黑客直接看到所有人的密码！
// bcrypt 是一种"单向哈希算法"：
// - 从密码可以算出哈希（加密）
// - 从哈希不能还原密码（单向）
// - 每次加密结果不同（加盐）
// ============================================================

// HashPassword 生成 bcrypt 密码哈希。
// 输入：明文密码（比如 "mypassword123"）
// 输出：加密后的哈希值（比如 "$2a$10$..."）
func HashPassword(password string) (string, error) {
	// bcrypt.GenerateFromPassword 把明文密码"搅碎"成哈希
	// bcrypt.DefaultCost = 10，表示加密的"强度"（越大越安全但越慢）
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 比对密码与哈希值。
// 用户登录时：输入密码 → 查数据库拿到哈希 → 用这个函数比对
func CheckPasswordHash(password, hash string) bool {
	// CompareHashAndPassword 把输入的密码和哈希对比
	// 如果匹配返回 nil（无错误）
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ============================================================
// 🎫 JWT（JSON Web Token）
//
// JWT 是一种"令牌"（token）格式，包含三部分：
// 1. Header（头部）：说明用了什么算法
// 2. Payload（负载）：携带的数据（用户名、角色、过期时间）
// 3. Signature（签名）：用密钥对整个令牌签名，防止篡改
//
// 一个 JWT 长这样：
// eyJhbGciOiJIUzI1NiJ9.eyJ1c2VybmFtZSI6ImFkbWluIn0.xxx
//
// 特点：
// - 无状态：服务器不需要存 session，令牌本身就包含所有信息
// - 防篡改：任何修改都会导致签名验证失败
// - 可过期：过期后自动失效，不需要服务器主动删除
// ============================================================

// Claims JWT 令牌中携带的自定义数据
// 除了标准的注册声明（过期时间、签发时间），
// 我们还额外存了用户名和角色
type Claims struct {
	Username string `json:"username"` // 👤 用户名
	Role     string `json:"role"`     // 👑 角色（admin/deployer/viewer）
	jwt.RegisteredClaims              // 📋 JWT 标准字段（过期时间、签发时间等）
}

// GenerateToken 为用户生成 JWT 令牌。
// 就像游乐园给你一个"手环"——上面写着你是谁、有什么权限、有效期到什么时候。
func GenerateToken(username, role, secret string, duration time.Duration) (string, error) {
	// 1. 构造令牌的"负载"（Claims）：把用户名、角色、时间等信息放进去
	claims := Claims{
		Username: username,                    // 👤 谁
		Role:     role,                        // 👑 什么角色
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // ⏰ 过期时间 = 现在 + 有效期
			IssuedAt:  jwt.NewNumericDate(time.Now()),               // 🕐 签发时间 = 现在
		},
	}

	// 2. 用 HS256 算法创建一个新的令牌（相当于选了一个"加密方式"）
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. 用密钥对令牌"签名"——就像在合同上盖章，防止被篡改
	// token.SignedString 返回最终的令牌字符串
	return token.SignedString([]byte(secret))
}

// ParseToken 校验并解析 JWT 令牌，成功时返回用户名和角色。
// 用户每次请求 API 时都会带上令牌，我们通过这个函数验证它是否有效。
func ParseToken(tokenStr, secret string) (string, string, error) {
	// 解析令牌：用密钥验证签名，然后把数据提取到 Claims 结构体中
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 安全检查：确认令牌使用的是我们预期的 HMAC 算法
		// 防止攻击者用其他算法（比如"none"算法）伪造令牌！
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil // 返回密钥，用于验证签名
	})

	if err != nil {
		// ❌ 令牌无效（过期了、被篡改了、格式不对）
		return "", "", err
	}

	// 提取令牌中的数据
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.Username, claims.Role, nil // ✅ 返回用户名和角色
	}

	// ❌ 令牌无法解析
	return "", "", errors.New("invalid token")
}

// ============================================================
// 🚪 Gin 中间件（Middleware）
//
// 中间件 = 在请求到达实际处理函数之前/之后，先经过的"关卡"。
// 就像进教学楼：
// 1. 先过门卫（AuthMiddleware）——检查有没有学生证
// 2. 再过闸机（RoleMiddleware）——检查能不能去实验室
// 3. 最后才能进教室（实际 API 处理函数）
// ============================================================

// AuthMiddleware Gin 的 JWT 鉴权中间件。
// 检查每个请求是否携带了有效的 JWT 令牌。
// 就像一个"门卫"——没带学生证？不准进！
func AuthMiddleware(secret string) gin.HandlerFunc {
	// 返回一个函数——这就是 Gin 中间件的标准写法
	return func(c *gin.Context) {
		// 从 HTTP 请求头中取出 Authorization（授权）字段
		// 格式通常是：Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")

		// 如果没带 Authorization 头，直接返回 401 未授权
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return // 注意：这里用 Abort，后面的处理函数不会再执行了！
		}

		// 解析 Authorization 头：按空格分割
		// 标准格式是 "Bearer xxx"，所以 parts[0]="Bearer", parts[1]=令牌内容
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
			return
		}

		// 解析令牌，取出用户名和角色
		username, role, err := ParseToken(parts[1], secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// ✅ 鉴权成功！
		// 把用户名和角色存到请求的上下文中
		// 这样后面的处理函数可以直接通过 c.Get("username") 拿到当前用户
		c.Set("username", username)
		c.Set("role", role)

		c.Next() // 继续执行下一个中间件或处理函数
	}
}

// RoleMiddleware 基于 RBAC 的角色访问控制中间件。
//
// RBAC = Role-Based Access Control（基于角色的访问控制）
// 简单说：不同角色能做不同的事
// - admin（管理员）：什么都能做
// - deployer（部署者）：可以部署项目
// - viewer（查看者）：只能看，不能操作
//
// @Ref: docs/sps/plans/20260527_nanoplan_m2_rbac_webhooks.md
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中取出角色（上面 AuthMiddleware 放进去的）
		roleInter, exists := c.Get("role")
		if !exists {
			// 没有角色信息？说明没经过 AuthMiddleware！这是程序 bug
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found in context"})
			return
		}

		role, ok := roleInter.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid role format"})
			return
		}

		// 判断当前用户的角色是否在"允许的角色列表"中
		allowed := false
		for _, r := range allowedRoles {
			if role == r {
				allowed = true
				break
			}
		}

		if !allowed {
			// ❌ 权限不足！比如 viewer 想操作部署，被拦住了
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			return
		}

		c.Next() // ✅ 有权限，放行！
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: bcrypt 加密和普通加密有什么区别？
//    A: bcrypt 是"单向加密"——加密后不能解密回去！
//       而且每次加密同一密码，结果都不同（加了随机"盐"）。
//       这样就算数据库泄露，黑客也拿不到原始密码~
//
// 2. Q: JWT 令牌的三个部分分别是什么？
//    A: Header（我用了什么算法）+ Payload（我是谁）+ Signature（防伪章）
//       就像：护照封面 + 个人信息页 + 防伪水印~
//
// 中级（面试常考）：
// 3. Q: JWT 和传统 Session 有什么区别？
//    A: Session 存在服务器（需要占用服务器内存），
//       JWT 存在客户端（服务器验证签名即可）。
//       JWT 适合分布式系统，不需要共享 session 存储~
//
// 4. Q: 为什么 ParseToken 里要检查 SigningMethodHMAC？
//    A: 防止"算法混淆攻击"！攻击者可能把算法改成 "none"，
//       然后伪造任意令牌。检查算法类型确保用的是我们指定的 HMAC~
//
// 5. Q: c.Abort() 和 c.Next() 有什么区别？
//    A: Abort = 拦截请求，不再执行后续中间件和处理函数
//       Next = 放行请求，继续执行后续中间件
//       就像安检：查出问题就 Abort（不让进），没问题就 Next（请进）~
//
// 高级（架构师级别）：
// 6. Q: RBAC 模型有什么优缺点？
//    A: 优点：简单直观，适合大多数场景
//       缺点：角色多了会很复杂（角色爆炸），
//       更细粒度的权限可能需要 ABAC（基于属性的访问控制）~
// ============================================================
