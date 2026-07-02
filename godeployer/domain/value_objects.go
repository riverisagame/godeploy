// ============================================================
// 文件：value_objects.go
// 作用：🎨 定义系统中所有的"值对象"（Value Object）！
//
// 什么是"值对象"？
// 值对象和实体（Entity）的区别：
// - 实体有"身份"（ID）：比如两个叫"张三"的人，身份证号不同就是不同人
// - 值对象没有"身份"：比如"红色"这个颜色，不管在哪都是红色
//   "100 元"这个金额，不管是谁的 100 元，价值都一样
//
// 这个文件定义了 DeployStatus（部署状态）——一个典型的值对象！
// 部署状态就是"字符串"本身，pending 就是 pending，没有身份的概念。
//
// 给初二小白的比喻：
// 想象一下交通灯的颜色🚦：
// - 红色 = 停止（StatusFailed）
// - 绿色 = 通行（StatusSuccess）
// - 黄色 = 等待（StatusPending）
// 颜色本身就是一个"值"——红色就是红色，不用问"这个红色是谁的"~
// ============================================================

// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31

package domain

// ============================================================
// 🎯 DeployStatus：部署任务状态
//
// type DeployStatus string —— 这是 Go 的"类型别名"
// 意思就是：DeployStatus 本质上还是一个字符串，
// 但我们把它包装成"新类型"，这样其他字符串不能随便当状态用！
//
// 好处：
// 1. 类型安全：函数参数写 DeployStatus，不能随便传一个 "abc" 进去
// 2. 集中管理：所有可能的状态值都在这里定义，一目了然
// 3. 附带方法：可以给 DeployStatus 加方法，比如 Valid()、IsTerminal()
// ============================================================

// DeployStatus 部署任务状态值对象。
// 预定义合法状态，禁止外部直接构造无效状态字符串。
type DeployStatus string

// 🎉 下面就是所有可能的状态值！
// 像不像游戏里的成就系统？🏆
const (
	StatusPending            DeployStatus = "pending"                 // ⏳ 等待中：任务刚创建，还没开始部署
	StatusDeploying          DeployStatus = "deploying"               // 🚀 部署中：正在进行部署操作
	StatusSuccess            DeployStatus = "success"                 // ✅ 成功了：部署顺利完成！
	StatusFailed             DeployStatus = "failed"                  // ❌ 失败了：哪里出了问题
	StatusAborted            DeployStatus = "aborted"                 // 🚫 已取消：用户主动中止了部署
	StatusRolledBack         DeployStatus = "rolled_back"             // ↩️ 已回滚：部署后发现问题，退回上一版本
	StatusFailedLockRejected DeployStatus = "failed_lock_rejected"    // 🔒 被拒绝：同时有人部署同一个项目，后来的被拒了
	StatusCriticalBrainSplit DeployStatus = "critical_brain_split"    // ☠️ 脑裂！：部分服务器回滚成功、部分失败——最严重的问题！
)

// Valid 判断当前状态是不是上面定义的"合法状态"之一。
// 就像一个门禁系统：只有名单上的人才能进入~
func (s DeployStatus) Valid() bool {
	switch s {
	case StatusPending, StatusDeploying, StatusSuccess, StatusFailed,
		StatusAborted, StatusRolledBack, StatusFailedLockRejected, StatusCriticalBrainSplit:
		return true // ✅ 合法状态
	}
	return false // ❌ 非法状态（哪来的野鸡状态？）
}

// IsTerminal 判断当前状态是不是"终态"——也就是不会再变的状态了！
// 就像人的状态：活着（可以继续变化）vs 终态（不能再变了）
// 成功、失败、取消——这三个状态是"最终结果"，不会再变成别的
func (s DeployStatus) IsTerminal() bool {
	return s == StatusSuccess || s == StatusFailed || s == StatusAborted
}

// IsRunnable 判断当前状态能不能"启动部署"。
// 只有"等待中"（pending）的任务才能启动
// 就像你只能从起点开始跑步，不能从终点开始跑~
func (s DeployStatus) IsRunnable() bool {
	return s == StatusPending
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 值对象（Value Object）和实体（Entity）有什么区别？
//    A: 实体有唯一 ID（比如每个人的身份证号），
//       值对象只有"值"本身（比如"红色"就是红色，不需要 ID）~
//
// 2. Q: type DeployStatus string 是什么意思？
//    A: 给 string 类型起了一个新名字叫 DeployStatus。
//       虽然本质还是字符串，但不能随便把普通字符串当状态用了！
//
// 中级（面试常考）：
// 3. Q: 为什么用 const 定义状态常量？
//    A: 1) 防止拼写错误（写 StatusSuccess 比写 "success" 安全）
//       2) 集中管理，修改时只改一处
//       3) IDE 自动补全，写代码更快~
//
// 4. Q: 什么是"终态"（Terminal State）？
//    A: 到了这个状态就不会再变了。success/failed/aborted 就是终态，
//       否则可能会出现"已经成功的任务又变成失败了"这种荒谬情况~
//
// 高级（架构师级别）：
// 5. Q: StatusCriticalBrainSplit（脑裂）是什么意思？
//    A: 分布式系统中最可怕的问题！当多台服务器需要同时切换版本，
//       一部分成功一部分失败时，整个系统处于"分裂"状态——
//       不同服务器运行不同版本，可能导致数据不一致！
//       这个状态专门标记这种极端异常情况~
//
// 6. Q: 为什么把状态定义为值对象，而不是直接在实体中用字符串？
//    A: 类型安全 + 行为封装。普通字符串允许任何值（包括无效值），
//       而 DeployStatus 可以附带 Valid()、IsTerminal() 等方法，
//       让状态的合法性检查集中在一处，而不是散落在各处~
// ============================================================
