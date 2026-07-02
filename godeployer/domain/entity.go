// ============================================================
// 文件：entity.go
// 作用：🧱 定义系统中所有"核心概念"的数据结构！
//
// 什么是 entity（实体）？
// 实体 = 有唯一身份的东西。就像每个人都有身份证号一样，
// 程序中每个实体对象也有自己的 ID，可以区分"谁是谁"。
//
// 这个文件定义了整个 GoDeploy 系统中最重要的"积木块"：
// - 配置（Config）：系统的各种设置
// - 项目（ProjectConfig）：你要部署的代码项目
// - 环境（EnvironmentConfig）：测试环境、生产环境
// - 服务器（ServerConfig）：目标服务器的连接信息
// - 部署任务（DeployTask）：一次部署操作的记录
// - 用户（UserResponse）：登录系统的用户
//
// 给初二小白的比喻：
// 如果你在玩 Minecraft，实体就是：
// - 你建的房子 = 项目配置
// - 房子的位置 = 服务器地址
// - 建房子的动作 = 部署任务
// - 你自己 = 用户
// ============================================================

package domain

// 📦 需要引入的外部工具
import (
	"context" // 🎯 上下文：用来控制"超时取消"，比如"5 秒内必须完成"
	"errors"  // ❌ 错误处理：创建自定义错误信息
	"time"    // ⏰ 时间：记录任务创建时间等
)

// ============================================================
// 🎯 DeployJob：部署任务"实体"
//
// 当你点击"开始部署"按钮时，系统会创建一个 DeployJob。
// 它描述了一次部署任务的所有信息：
// - 做什么（TaskID）
// - 用什么配置（Config）
// - 日志写到哪里（LogFilePath）
// - 如果用户取消，怎么通知（Ctx, Cancel）
//
// 就像一张"派工单"：写着工人编号、施工图纸、工作日志本~
// ============================================================

// DeployJob 部署任务实体
type DeployJob struct {
	Ctx         context.Context       // 📡 控制信号通道：用来通知"停止工作"
	Cancel      context.CancelFunc    // 🛑 取消函数：调用 = 告诉工人"别干了"
	TaskID      int64                 // 🔖 任务唯一编号（在数据库中的 ID）
	Config      *Config               // 📋 系统配置（项目、环境、服务器信息等）
	LogFilePath string                // 📝 日志文件路径（部署过程中的输出写到这里）
}

// NewDeployJob 创建部署任务实体，初始化 Context 和 CancelFunc。
// 就像一个"派工单"工厂：你给我原材料（taskID, config），我帮你做好派工单！
func NewDeployJob(taskID int64, config *Config, logFilePath string) *DeployJob {
	// context.WithCancel 创建一个"可取消的上下文"
	// 它返回两个东西：
	// 1. ctx：上下文本身，带着一个"监听器"
	// 2. cancel：一个按钮，按下去就会通知 ctx "结束了！"
	ctx, cancel := context.WithCancel(context.Background())

	// &DeployJob{...} 表示创建一个新的 DeployJob 对象
	// & 符号的意思是"取地址"，就像说"这个人的家庭住址"
	return &DeployJob{
		Ctx:         ctx,         // 📡 信号监听器
		Cancel:      cancel,      // 🛑 取消按钮
		TaskID:      taskID,      // 🔖 任务编号
		Config:      config,      // 📋 配置信息
		LogFilePath: logFilePath, // 📝 日志路径
	}
}

// IsCancelled 判断任务是否已被取消。
// 想象你在打游戏：如果妈妈喊你吃饭（取消信号），你的游戏就"被取消"了。
// 这个函数就是检查：妈妈喊我了吗？
func (j *DeployJob) IsCancelled() bool {
	// select 是 Go 独有的"多路选择器"，哪个通道有消息就处理哪个
	select {
	// <-j.Ctx.Done() 意思是：监听"取消"通道
	// 如果通道关闭了（收到消息），说明有人调用了 Cancel()
	case <-j.Ctx.Done():
		return true // ✅ 确实被取消了
	default:
		return false // ❌ 还没人取消，继续干活！
	}
}

// ============================================================
// ⚙️ Config：整个系统的配置信息
//
// 配置文件是 YAML 格式（一种比 JSON 更好写的数据格式）。
// 这个结构体就是 YAML 文件的"镜像"，程序把 YAML 读进来，
// 就变成了这个结构体里的数据。
//
// 结构体（struct）：Go 语言中把多个字段组合在一起的方式
// "yaml:xxx" 的标签：告诉 Go 解析 YAML 时，字段名对应什么
// ============================================================

// Config 配置相关的实体
type Config struct {
	Global           GlobalConfig             `yaml:"global"`             // 🌍 全局配置（端口、数据库路径等）
	ProjectConfigDir string                   `yaml:"project_config_dir"` // 📂 项目配置文件夹路径
	Projects         map[string]ProjectConfig `yaml:"-"`                  // 📋 所有项目配置（yaml:"-" 表示不从 YAML 直接读，由 Loader 组装）
}

// GlobalConfig 全局设置——相当于"系统设置"页面
type GlobalConfig struct {
	SQLitePath     string `yaml:"sqlite_path"`     // 💾 SQLite 数据库文件存哪里（比如 ./deploy.db）
	LogPath        string `yaml:"log_path"`        // 📝 日志文件存哪里
	WorkspacePath  string `yaml:"workspace_path"`  // 🏗️ 工作区：从 Git 拉下来的代码放哪里
	SSHKeyPath     string `yaml:"ssh_key_path"`    // 🔑 SSH 私钥路径（用来登录远程服务器）
	ServerPort     int    `yaml:"server_port"`     // 🚪 HTTP 服务监听端口（比如 8080）
	JWTSecret      string `yaml:"jwt_secret"`      // 🔐 JWT 密钥（用来签发登录令牌，就像游乐场的手环）
	DiffMaxSizeKB  int    `yaml:"diff_max_size_kb"` // 📏 代码差异（diff）最大保存多少 KB
	DiskMinSpaceMB int    `yaml:"disk_min_space_mb"` // 💿 磁盘最少剩多少 MB，不够就不写 diff 了
	TaskRetainMax  int    `yaml:"task_retain_max"`   // 📊 每个环境最多保留多少条任务记录
	TaskRetainDays int    `yaml:"task_retain_days"`  // 📅 任务记录保留多少天
}

// ProjectConfig 一个"项目"的配置——比如你的 ThinkPHP 博客系统、Vue 官网等
type ProjectConfig struct {
	ID            string              `yaml:"id" json:"id"`                          // 🆔 项目唯一标识，比如 "myblog"
	Name          string              `yaml:"name" json:"name"`                      // 📛 项目显示名称，比如 "我的博客"
	Repo          string              `yaml:"repo" json:"repo"`                      // 🌐 Git 仓库地址，比如 git@github.com:me/myblog.git
	WebhookSecret string              `yaml:"webhook_secret" json:"webhook_secret"`   // 🤫 Webhook 密钥（GitHub 推送通知时验证身份用）
	Branch        string              `yaml:"branch" json:"branch"`                  // 🌿 默认分支，比如 "main" 或 "master"
	Exclude       []string            `yaml:"exclude" json:"exclude"`                // 🚫 同步到服务器时要排除的文件/目录，比如 node_modules
	SharedFiles   []string            `yaml:"shared_files" json:"shared_files"`       // 🔗 多个版本之间"共享"的文件（比如 .env 配置文件）
	SharedDirs    []string            `yaml:"shared_dirs" json:"shared_dirs"`         // 📁 多个版本之间"共享"的目录（比如 uploads 上传文件）
	Build         BuildConfig         `yaml:"build" json:"build"`                     // 🏗️ 构建配置（部署前要执行的命令）
	Environments  []EnvironmentConfig `yaml:"environments" json:"environments"`       // 🌍 环境列表（测试、生产等）
}

// BuildConfig 部署前的"构建"步骤——就是在服务器上跑一些命令来编译代码
// 比如：npm install && npm run build
type BuildConfig struct {
	BeforeSync []string `yaml:"before_sync" json:"before_sync"` // 📋 同步前要执行的命令列表（按顺序执行）
}

// EnvironmentConfig 一个"环境"的配置
// 环境 = 代码运行的地方。通常有：
// - testing（测试环境）：开发人员自己测试用
// - staging（预发布环境）：上线前最后的验证
// - production（生产环境）：真正的用户在用
type EnvironmentConfig struct {
	ID            string         `yaml:"id" json:"id"`                          // 🆔 环境标识，比如 "staging" 或 "production"
	Name          string         `yaml:"name" json:"name"`                      // 📛 环境名称，比如 "测试环境"
	DefaultBranch string         `yaml:"default_branch" json:"default_branch"`   // 🌿 该环境默认部署的分支
	KeepReleases  int            `yaml:"keep_releases" json:"keep_releases"`     // 💾 保留最近多少次发布版本
	Servers       []ServerConfig `yaml:"servers" json:"servers"`                 // 🖥️ 这个环境包含哪些服务器
	BeforeSymlink []string       `yaml:"before_symlink" json:"before_symlink"`   // 🔗 切换软链接前要执行的命令
	AfterSymlink  []string       `yaml:"after_symlink" json:"after_symlink"`     // 🔗 切换软链接后要执行的命令
}

// ServerConfig 一台"目标服务器"的连接信息
// 就像你要去朋友家玩需要知道地址、门牌号一样~
type ServerConfig struct {
	Host       string `yaml:"host" json:"host"`               // 🌐 服务器 IP 地址或域名（如 192.168.1.100）
	Port       int    `yaml:"port" json:"port"`               // 🚪 SSH 端口（默认 22）
	User       string `yaml:"user" json:"user"`               // 👤 SSH 登录用户名
	DeployTo   string `yaml:"deploy_to" json:"deploy_to"`     // 📂 部署目录（代码要放到服务器哪个文件夹）
	SSHKeyPath string `yaml:"ssh_key_path" json:"ssh_key_path"` // 🔑 SSH 私钥路径（用来验证你是谁）
}

// ============================================================
// 👤 UserResponse：用户信息（包含数据库映射）
//
// gorm:"..." 标签：告诉 GORM（Go 的数据库工具）这个字段对应的数据库列
// json:"..." 标签：告诉程序这个字段在 JSON 传输时叫什么名字
// - primaryKey：主键（唯一标识一条记录）
// - autoIncrement：自动递增（每加一个用户，ID 自动 +1）
// - uniqueIndex：唯一索引（不能有两个相同用户名）
// - not null：不能为空
// ============================================================

// UserResponse 用户信息——谁在登录使用这个系统
type UserResponse struct {
	ID                 int       `json:"id" gorm:"primaryKey;autoIncrement"`       // 🆔 用户 ID（数据库自动生成）
	Username           string    `json:"username" gorm:"uniqueIndex;not null"`     // 📛 用户名（登录时用的）
	Role               string    `json:"role" gorm:"not null"`                     // 👑 角色身份（admin=管理员, deployer=部署者, viewer=围观者）
	CreatedAt          time.Time `json:"created_at" gorm:"not null"`               // 📅 账号创建时间
	BoundGitAuthors    string    `json:"bound_git_authors" gorm:"default:''"`       // 🔗 绑定的 Git 作者名（用于权限校验）
	RestrictGitAuthors bool      `json:"restrict_git_authors" gorm:"default:false"` // 🚫 是否限制 Git 作者
	PermittedProjects  string    `json:"permitted_projects" gorm:"default:'*'"`    // ✅ 允许操作的项目列表（* 表示全部）
	PasswordHash       string    `json:"-" gorm:"not null"`                        // 🔐 密码的哈希值（json:"-" 表示不显示在 JSON 里！）
}

// TableName 告诉 GORM：UserResponse 对应数据库的 users 表
func (UserResponse) TableName() string {
	return "users"
}

// ============================================================
// 📜 GitCommit：一次 Git 提交的信息
// 就像一次作业的"快照"——记录了谁、什么时间、改了什么东西
// ============================================================

// GitCommit 实体
type GitCommit struct {
	Hash      string `json:"hash"`       // 🆔 提交的 SHA 值（唯一标识这次提交）
	Message   string `json:"message"`    // 📝 提交信息（程序员写的改动说明）
	Author    string `json:"author"`     // 👤 作者
	CreatedAt string `json:"created_at"` // 📅 提交时间
}

// ============================================================
// 📋 DeployTask：一次部署任务的完整记录（数据库中的 deploy_tasks 表）
//
// 每次有人点击"部署"按钮，系统就会创建一条 DeployTask 记录。
// 它记录了：
// - 部署什么项目（ProjectID）
// - 部署到什么环境（EnvID）
// - 部署哪个版本（CommitID）
// - 现在进行到哪一步了（Status）
// - 哪个用户操作的（UserID, Username）
// - 什么时候干的（CreatedAt）
// ============================================================

// DeployTask 记录部署任务状态的实体（对应数据库中的 deploy_tasks 表）
type DeployTask struct {
	ID             int          `json:"id" gorm:"primaryKey;autoIncrement"`     // 🆔 任务 ID（主键，自动递增）
	ProjectID      string       `json:"project_id" gorm:"not null;index"`       // 📁 项目标识（比如 "myblog"）
	EnvID          string       `json:"env_id" gorm:"not null;index"`           // 🌍 环境标识（比如 "production"）
	CommitID       string       `json:"commit_id" gorm:"not null"`              // 🔖 要部署的 Git 提交 SHA
	Status         DeployStatus `json:"status" gorm:"not null;index"`           // 📊 当前状态（pending → deploying → success/failed）
	ReleaseName    string       `json:"release_name" gorm:"not null"`           // 📦 发布版本的目录名（比如 20260601_123456）
	UserID         int          `json:"user_id" gorm:"not null"`                // 👤 操作人的用户 ID
	Username       string       `json:"username" gorm:"not null"`               // 👤 操作人的用户名
	ConfigSnapshot string       `json:"config_snapshot" gorm:"not null"`        // 📸 部署时的配置快照（方便以后查历史）
	CreatedAt      time.Time    `json:"created_at" gorm:"not null"`             // 📅 创建时间
	Description    string       `json:"description" gorm:"default:''"`          // 💬 备注说明
	ExtraExclude   string       `json:"extra_exclude" gorm:"default:''"`        // 🚫 附加排除规则（逗号分隔）
	TargetType     string       `json:"target_type" gorm:"default:''"`          // 🎯 目标类型（commit=精确提交 / branch=分支 / tag=标签）
}

// TableName 告诉 GORM：DeployTask 对应数据库的 deploy_tasks 表
func (DeployTask) TableName() string {
	return "deploy_tasks"
}

// ============================================================
// 🔄 状态机：任务状态只能按固定规则变化！
//
// 就像红绿灯：绿灯→黄灯→红灯，不能绿灯直接变红灯。
// 任务状态的规则是：
//   pending（等待） →  deploying（部署中）
//   deploying（部署中） →  success（成功）、failed（失败）、aborted（取消）
//   其他情况 → 返回错误 ErrInvalidTransition
// ============================================================

// ErrInvalidTransition 当尝试非法状态转换时返回这个错误
// 比如：跳过 deploying，直接把任务从 pending 变成 success——不行！😠
var ErrInvalidTransition = errors.New("deploy task: invalid status transition")

// Start 把任务从"等待中"变成"部署中"
// 就像按下发射按钮——火箭从准备状态变成发射状态！
func (t *DeployTask) Start() error {
	// 只有 pending 状态才能开始部署
	if t.Status != StatusPending {
		return ErrInvalidTransition
	}
	t.Status = StatusDeploying // 🔄 状态变为"部署中"
	return nil
}

// Complete 把任务从"部署中"变成"成功"
// 就像火箭到达目的地，宣布任务圆满完成！
func (t *DeployTask) Complete() error {
	// 只有 deploying 状态才能成功
	if t.Status != StatusDeploying {
		return ErrInvalidTransition
	}
	t.Status = StatusSuccess // ✅ 部署成功
	return nil
}

// Fail 把任务从"部署中"变成"失败"
// 就像火箭中途爆炸了……😱
func (t *DeployTask) Fail() error {
	// 只有 deploying 状态才能失败
	if t.Status != StatusDeploying {
		return ErrInvalidTransition
	}
	t.Status = StatusFailed // ❌ 部署失败
	return nil
}

// Abort 把任务取消，支持从 pending（等待中）或 deploying（部署中）取消
// 就像说"计划取消，不干了！"
func (t *DeployTask) Abort() error {
	// 可以取消"等待中"或者"部署中"的任务
	if t.Status != StatusPending && t.Status != StatusDeploying {
		return ErrInvalidTransition
	}
	t.Status = StatusAborted // 🚫 已取消
	return nil
}

// IsActive 检查任务是否"活跃"——也就是还没干完
// 相当于问："这单活还在干不？"
func (t *DeployTask) IsActive() bool {
	// 如果是等待中或者部署中，都算活跃
	return t.Status == StatusPending || t.Status == StatusDeploying
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: struct（结构体）是什么？
//    A: 把多个相关数据打包成一个"复合类型"，就像学生档案：姓名+年龄+班级~
//
// 2. Q: gorm:"primaryKey;autoIncrement" 这种标签有什么用？
//    A: 告诉数据库：这个字段是主键（唯一标识），并且每次新增自动编号！
//
// 3. Q: json:"-" 是做什么的？
//    A: 序列化成 JSON 时忽略这个字段！密码哈希绝对不能暴露出去~
//
// 中级（面试常考）：
// 4. Q: context.Context 在 Go 中的作用？
//    A: 用来传递"控制信号"——比如超时、取消。多个 goroutine 共享同一个 context，
//       一个取消信号可以同时通知所有正在工作的协程~
//
// 5. Q: 为什么需要状态机（Start → Complete/Fail）？
//    A: 防止非法操作！比如一个已经成功的任务不能再次"完成"。
//       状态机保证数据的一致性和程序的健壮性~
//
// 6. Q: map[string]ProjectConfig 是什么数据类型？
//    A: 映射表（键值对）。key 是项目 ID（字符串），value 是对应的项目配置。
//       就像查字典：输入"myblog"就能找到 myblog 项目的配置~
//
// 高级（架构师级别）：
// 7. Q: 为什么要用状态机模式管理部署任务状态？
//    A: 1) 防止状态回跳（已经成功的任务不能回到部署中）
//       2) 每个状态转换都有明确的规则，方便审计
//       3) 为后续添加"状态监听器"（状态变化时触发通知）打下基础~
//
// 8. Q: 配置用 YAML 而不是 JSON 的原因？
//    A: YAML 支持注释、更可读，适合人工编写和维护的配置文件~
// ============================================================
