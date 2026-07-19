package application

import "pdeploy/internal/domain"

// SSHClient 远程服务器命令执行与文件同步接口
type SSHClient interface {
	// RunCommand 在远程服务器上执行命令，输出流式写入 logChan
	RunCommand(server *domain.Server, cmd string, logChan chan<- string) error
	// SyncFiles 将本地目录同步到远程服务器指定路径
	SyncFiles(server *domain.Server, localPath, remotePath string, logChan chan<- string) error
}

// GitClient 本地 Git 操作接口
type GitClient interface {
	// CloneOrPull 拉取或更新代码到本地 workspace，返回 workspace 路径
	CloneOrPull(repoURL, branch, projectName string, logChan chan<- string) (workspacePath string, err error)
}
