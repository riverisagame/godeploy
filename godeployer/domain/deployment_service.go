// @Ref: docs/sps/specs/20260531-ddd-full-tactical-design.md | @Date: 2026-05-31
package domain

import "context"

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

// DeploymentService 编排两阶段部署流程的领域服务。
// 封装"并行 Rsync → 统一切换软链"的两阶段提交策略。
type DeploymentService struct {
	taskRepo TaskRepository
}

// NewDeploymentService 创建部署领域服务实例。
func NewDeploymentService(taskRepo TaskRepository) *DeploymentService {
	return &DeploymentService{taskRepo: taskRepo}
}

// ShouldRollback 判断部署失败后是否需要回滚到上一个成功版本。
func (s *DeploymentService) ShouldRollback(task *DeployTask) bool {
	return task.Status == StatusFailed
}
