// ============================================================
// 文件：deployment_service.go
// 作用：🧠 定义"部署操作"的核心规则！
//
// 这个文件定义了"部署"这件事的抽象接口。
// 什么叫"抽象接口"？就像外卖App上的"点餐"按钮——
// 你按按钮，厨房就开始做菜，但你不用管厨房怎么运作的。
//
// 这里定义了 3 个核心操作：
// 1. Rsync：把代码传到远程服务器（就像寄快递）
// 2. SwitchSymlink：切换软链接实现"零停机更新"（就像换路牌）
// 3. RunCommand：在远程服务器上执行命令（就像远程遥控）
//
// 它还定义了"两阶段提交"策略：
// 第一阶段：多台服务器同时接收代码
// 第二阶段：所有服务器一起切换（如果有一台失败就全部回滚）
//
// 给初二小白的比喻：
// 部署就像"搬家"：
// Phase 1 = 把所有家具搬进新家（各搬各的）
// Phase 2 = 所有人同时宣布"我们搬好了！"（失败就全部搬回去）
// ============================================================

// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31
// 这行是项目内部的"引用标记"——指向这份代码的设计文档
// @Ref 告诉开发者："想知道为什么这么设计？去看这个文档！"
// @Date 记录了这个设计的时间

package domain

import "context"

// ============================================================
// 🎯 NodeExecutor：节点执行器接口（Interface）
//
// Interface（接口）是 Go 语言中最重要的概念之一！
// 它定义了一组"方法的契约"——说好要做什么，但不规定怎么做。
//
// 就像你去饭店点餐：
// - 菜单（接口）写着"宫保鸡丁"——说好了有这道菜
// - 但不同厨师（不同实现）做出来的味道可能不同
// - 你只关心能不能吃到，不关心厨师怎么炒
//
// 这里的 NodeExecutor 接口定义了"对一台服务器能做什么操作"：
// - Rsync：发文件
// - SwitchSymlink：切软链
// - RunCommand：跑命令
// - Close：关闭连接
//
// 为什么要有接口？
// 因为在"领域层"只需要知道"我能操作远程服务器"就够了，
// 至于怎么操作（SSH？本地直接执行？模拟测试？），
// 那是"基础设施层"的事——通过接口解耦！
// ============================================================

// NodeExecutor 定义部署节点操作接口。
// domain 层定义此接口，infrastructure 层实现——依赖反转原则。
type NodeExecutor interface {
	// Rsync 将构建产物同步到目标节点。excludes 为排除模式列表，linkDest 为硬链接参考目录。
	Rsync(ctx context.Context, node ServerConfig, localPath, remotePath, linkDest string, excludes []string) error

	// SwitchSymlink 在目标节点上执行原子软链接切换。
	SwitchSymlink(ctx context.Context, node ServerConfig, releaseName string) error

	// RunCommand 在目标节点上执行任意命令，返回标准输出。
	RunCommand(ctx context.Context, node ServerConfig, cmd string) (string, error)

	// Close 释放所有连接资源。
	Close() error
}

// ============================================================
// 📋 DeploymentService：部署领域服务
//
// 什么是"领域服务"？
// 就是"跟业务逻辑相关的操作"——不属于某个具体对象，而是跨越多个对象的操作。
//
// 比如"两阶段提交"：它同时涉及多台服务器、多个 release 版本，
// 无法放在某一种实体（如 DeployTask）里，所以单独作为一个"服务"。
//
// 这里目前比较简单，只有一个 ShouldRollback 方法，
// 但未来可以扩展出更多部署相关的核心逻辑。
// ============================================================

// DeploymentService 编排两阶段部署流程的领域服务。
// 封装"并行 Rsync → 统一切换软链"的两阶段提交策略。
type DeploymentService struct {
	taskRepo TaskRepository // 📚 任务仓库：用来查询和操作部署任务
}

// NewDeploymentService 创建部署领域服务实例。
// 就像一个"构造函数"：你给它一个 taskRepo，它给你一个能用的部署服务。
func NewDeploymentService(taskRepo TaskRepository) *DeploymentService {
	return &DeploymentService{
		taskRepo: taskRepo, // 把"任务仓库"存起来，以后要用
	}
}

// ShouldRollback 判断部署失败后是否需要回滚到上一个成功版本。
// 当前逻辑很简单：只要任务状态是 failed，就应该回滚。
//
// 为什么要把这个判断放在领域层？
// 因为"什么情况下需要回滚"是业务规则！未来可能：
// - 只有生产环境需要回滚，测试环境不需要
// - 只有特定类型的失败才回滚
// - 超过 50% 节点失败才回滚
// 把规则放在这里，改规则时只改这个文件就行了！
func (s *DeploymentService) ShouldRollback(task *DeployTask) bool {
	return task.Status == StatusFailed
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: interface（接口）是什么？
//    A: 像一份"合同"或"菜单"，说好了要提供哪些功能（方法），
//       但不规定具体怎么实现。厨师（不同的实现者）可以有自己的做法~
//
// 2. Q: context.Context 为什么重要？
//    A: 它可以携带"取消信号"！如果用户取消部署，context 会通知所有正在工作的协程停止~
//
// 中级（面试常考）：
// 3. Q: 什么是"依赖反转原则"（Dependency Inversion Principle）？
//    A: 高层模块不依赖低层模块的具体实现，而是依赖接口。
//       这里 domain 层定义 NodeExecutor 接口，infrastructure 层来实现它。
//       好处：换 SSH 实现方式（比如换库）时，领域层代码完全不用改！
//
// 4. Q: 什么是"领域服务"（Domain Service）？
//    A: 不属于某个具体实体的业务逻辑。比如"两阶段提交"涉及多台服务器、多个步骤，
//       不适合放在 DeployTask 或 ServerConfig 里，所以独立成服务~
//
// 高级（架构师级别）：
// 5. Q: 为什么 ShouldRollback 要放在 domain 层而不是 application 层？
//    A: "什么情况下需要回滚"是业务规则，属于领域逻辑。
//       放在 domain 层意味着：换任何 UI（Web 端/命令行/API），回滚规则都不变。
//       这是 DDD（领域驱动设计）的核心思想：业务规则永远在中心层~
// ============================================================
