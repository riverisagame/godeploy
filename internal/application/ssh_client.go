package application

import "pdeploy/internal/domain"

// SSHClient 远程服务器命令执行与文件同步接口
type SSHClient interface {
	// RunCommand 在远程服务器上执行命令，输出流式写入 logChan
	RunCommand(server *domain.Server, cmd string, logChan chan<- string) error
	// SyncFiles 将本地目录同步到远程服务器指定路径
	SyncFiles(server *domain.Server, localPath, remotePath, linkDest string, logChan chan<- string) error
}

// GitClient 本地 Git 操作接口
type GitClient interface {
	// CloneForDeploy prepares a deployment workspace using a git bare repo and worktree.
	CloneForDeploy(repoURL, branch, projectName string, deployID uint, logChan chan<- string) (workspacePath string, err error)
	// CleanupDeploy removes the temporary worktree after deployment finishes.
	CleanupDeploy(projectName string, deployID uint, deployPath string) error
	// FetchAndGetCommits 拉取最新代码并获取增量提交记录
	FetchAndGetCommits(repoURL, branch, projectName, fromCommit string) ([]domain.CommitInfo, error)
}
