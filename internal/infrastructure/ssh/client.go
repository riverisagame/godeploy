package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"pdeploy/internal/domain"
	"runtime"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client 实现 application.SSHClient 接口
type Client struct{}

// NewClient 创建 SSH 客户端
// @Ref: docs/sps/plans/20260719_p0_deploy_gaps_plan.md S2 | @Date: 2026-07-19
func NewClient() *Client {
	return &Client{}
}

// buildConfig 根据 Server 动态构建 SSH 配置
func (c *Client) buildConfig(server *domain.Server) (*ssh.ClientConfig, error) {
	user := server.User
	if user == "" {
		user = "root"
	}

	var authMethods []ssh.AuthMethod

	// 确定密钥路径：优先使用 Server.KeyPath，否则回退到默认
	keyPath := server.KeyPath
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			keyPath = filepath.Join(home, ".ssh", "id_rsa")
		}
	}

	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	var hostKeyCallback ssh.HostKeyCallback
	// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 1.2 | @Date: 2026-07-21
	home, err := os.UserHomeDir()
	if err == nil {
		knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
		if cb, err := knownhosts.New(knownHostsPath); err == nil {
			hostKeyCallback = cb
		}
	}
	if hostKeyCallback == nil {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}
	return config, nil
}

// RunCommand 在远程服务器执行命令，输出流式写入 logChan
func (c *Client) RunCommand(server *domain.Server, cmd string, logChan chan<- string) error {
	config, err := c.buildConfig(server)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", server.IP, server.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Failed to dial %s: %v\n", addr, err)
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Failed to create session: %v\n", err)
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	go streamOutput(stdout, logChan)
	go streamOutput(stderr, logChan)

	logChan <- fmt.Sprintf("[SSH] Executing on %s@%s: %s\n", server.User, server.IP, cmd)
	err = session.Run(cmd)
	if err != nil {
		logChan <- fmt.Sprintf("[SSH] Command failed: %v\n", err)
		return err
	}

	logChan <- fmt.Sprintf("[SSH] Command successful on %s\n", server.IP)
	return nil
}

// SyncFiles 使用 rsync 将本地目录同步到远程服务器
func (c *Client) SyncFiles(server *domain.Server, localPath, remotePath, linkDest string, logChan chan<- string) error {
	if runtime.GOOS == "windows" {
		// Windows 下无 rsync，使用 scp 回退方案
		logChan <- "[Sync] Windows detected, using scp fallback...\n"
		return c.scpFallback(server, localPath, remotePath, logChan)
	}

	user := server.User
	if user == "" {
		user = "root"
	}

	// 构建 rsync 命令
	sshCmd := fmt.Sprintf("ssh -p %d -o StrictHostKeyChecking=no", server.Port)
	if server.KeyPath != "" {
		sshCmd += fmt.Sprintf(" -i '%s'", server.KeyPath)
	}

	// @Ref: docs/sps/plans/20260721_production_fix_ir.md Task 1.3 | @Date: 2026-07-21
	rsyncArgs := []string{
		"-avz", "--delete", "--protect-args",
		"--exclude", ".git",
		"-e", sshCmd,
	}

	// 优化点：支持增量硬链接部署
	if linkDest != "" {
		rsyncArgs = append(rsyncArgs, fmt.Sprintf("--link-dest=%s", linkDest))
	}

	rsyncArgs = append(rsyncArgs, 
		localPath+"/", 
		fmt.Sprintf("%s@%s:%s/", user, server.IP, remotePath),
	)

	logChan <- fmt.Sprintf("[Sync] rsync %s -> %s@%s:%s\n", localPath, user, server.IP, remotePath)

	cmd := exec.Command("rsync", rsyncArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("[Sync] rsync failed to start: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		logChan <- "[Sync] " + scanner.Text() + "\n"
	}

	return cmd.Wait()
}

// scpFallback Windows 下使用 scp 替代 rsync
func (c *Client) scpFallback(server *domain.Server, localPath, remotePath string, logChan chan<- string) error {
	user := server.User
	if user == "" {
		user = "root"
	}

	scpArgs := []string{
		"-r", "-P", fmt.Sprintf("%d", server.Port),
		"-o", "StrictHostKeyChecking=no",
	}
	if server.KeyPath != "" {
		scpArgs = append(scpArgs, "-i", server.KeyPath)
	}
	scpArgs = append(scpArgs, localPath, fmt.Sprintf("%s@%s:'%s'", user, server.IP, remotePath))

	logChan <- fmt.Sprintf("[Sync] scp %s -> %s@%s:%s\n", localPath, user, server.IP, remotePath)

	cmd := exec.Command("scp", scpArgs...)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		logChan <- "[Sync] " + string(out) + "\n"
	}
	return err
}

func streamOutput(r io.Reader, logChan chan<- string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		logChan <- scanner.Text() + "\n"
	}
}
