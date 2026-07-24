package application

import (
	"context"

	"github.com/riverisagame/godeploy/internal/domain"
)

// SSHClient 远程服务器命令执行与文件同步接口
type SSHClient interface {
	// RunCommand 在远程服务器上执行命令，输出流式写入 logChan
	RunCommand(ctx context.Context, server *domain.Server, cmd string, logChan chan<- string) error
	// SyncFiles 将本地目录同步到远程服务器指定路径
	SyncFiles(ctx context.Context, server *domain.Server, localPath, remotePath, linkDest string, logChan chan<- string) error
}

// GitClient 本地 Git 操作接口
type GitClient interface {
	CloneForDeploy(ctx context.Context, repoURL, branch, projectName string, deployID uint, logChan chan<- string) (string, error)
	CleanupDeploy(ctx context.Context, projectName string, deployID uint, deployPath string) error
	FetchAndGetCommits(ctx context.Context, repoURL, branch, projectName, fromHash string) ([]domain.CommitInfo, error)
}
