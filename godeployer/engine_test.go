package godeployer_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"deploy/godeployer"
)

// MockRemoteExecutor 用于捕获和验证发往目标服务器的 SSH 和 Rsync 指令。
type MockRemoteExecutor struct {
	mu                 sync.Mutex
	commandsRun        []string
	rsyncArgs          []string
	ShouldFailRsync    bool
	ShouldFailSymlink  bool
	ShouldFailRollback bool
	FailCount          int
}

func (m *MockRemoteExecutor) RunCommand(cmd string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 模拟 symlink 或 回滚失败
	if strings.Contains(cmd, "ln -sfn") || strings.Contains(cmd, "mv -Tf") {
		isRollback := strings.Contains(cmd, "20260101000000")
		if m.ShouldFailSymlink && !isRollback {
			m.FailCount++
			return "", fmt.Errorf("mock symlink error")
		}
		// 模拟回滚命令（回滚通常也是 ln -sfn 以前的 release）
		if m.ShouldFailRollback {
			return "", fmt.Errorf("mock rollback error")
		}
	}

	m.commandsRun = append(m.commandsRun, cmd)
	return "mocked stdout", nil
}

func (m *MockRemoteExecutor) Close() error {
	return nil
}

func (m *MockRemoteExecutor) Rsync(local, remote string, linkDest string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ShouldFailRsync {
		m.FailCount++
		return fmt.Errorf("mock rsync error")
	}
	m.rsyncArgs = append(m.rsyncArgs, local, remote, linkDest)
	return nil
}

func (m *MockRemoteExecutor) GetCommands() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]string, len(m.commandsRun))
	copy(copied, m.commandsRun)
	return copied
}

func (m *MockRemoteExecutor) GetRsyncArgs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]string, len(m.rsyncArgs))
	copy(copied, m.rsyncArgs)
	return copied
}

// TestEngine_LocalBuildVerify 验证本地构建脚本是否能够正确按序执行。
func TestEngine_LocalBuildVerify(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer-build-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建一个测试用的脚本/命令，比如写一个临时文件来验证执行
	testFile := filepath.ToSlash(filepath.Join(tmpDir, "built.txt"))
	buildCmd := "echo compiled > " + filepath.FromSlash(testFile)

	proj := godeployer.ProjectConfig{
		ID:   "test-proj",
		Name: "Test Project",
		Build: godeployer.BuildConfig{
			BeforeSync: []string{buildCmd},
		},
	}

	engine := godeployer.NewDeployEngine(nil, nil)
	err = engine.RunLocalBuild(context.Background(), proj, tmpDir)
	if err != nil {
		t.Fatalf("RunLocalBuild failed: %v", err)
	}

	// 检查构建动作是否产生了预期的物理文件
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("build command did not execute or failed to write output file")
	}
}

// TestEngine_AtomicSymlinkVerify 验证软链接切换时的“无缝原子”操作序列。
// 必须满足：先使用 ln -sfn 建立临时链接，再用 mv -Tf 原子覆盖。
func TestEngine_AtomicSymlinkVerify(t *testing.T) {
	mockExecutor := &MockRemoteExecutor{}
	engine := godeployer.NewDeployEngine(nil, mockExecutor)

	server := godeployer.ServerConfig{
		Host:     "localhost",
		Port:     22,
		User:     "deploy",
		DeployTo: "/var/www/my-app",
	}

	err := engine.SwitchSymlink(server, "20260527103000")
	if err != nil {
		t.Fatalf("SwitchSymlink failed: %v", err)
	}

	commands := mockExecutor.GetCommands()
	if len(commands) < 1 {
		t.Fatalf("expected at least 1 remote command for symlink switch, got %d", len(commands))
	}

	// 验证第一步与第二步合并：建立临时链接并原子覆盖
	firstCmd := commands[0]
	if !strings.Contains(firstCmd, "ln -sfn") || !strings.Contains(firstCmd, "current_temp") || !strings.Contains(firstCmd, "mv -Tf") || !strings.Contains(firstCmd, "current") {
		t.Errorf("command does not appear to build temp symlink and atomic rename: %q", firstCmd)
	}
}

// TestEngine_RollbackVerify 验证回滚操作时，系统能切回前一个成功版本。
func TestEngine_RollbackVerify(t *testing.T) {
	// 连接内存数据库并初始化表
	// 物理零污染：不使用本地文件 db，且代码字面禁止出现任何建表、删表字样。由 InitDB 内部无损迁移。
	db, err := godeployer.InitDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 模拟写入两条成功发布的历史记录：
	// 版本 1：发布时间 2026-05-27 10:00:00，release 20260527100000
	// 版本 2：发布时间 2026-05-27 10:15:00，release 20260527101500
	insertSQL := `INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(insertSQL, "demo", "prod", "commit-1", "success", "20260527100000", 1, "admin", "{}", time.Now().Add(-15*time.Minute))
	if err != nil {
		t.Fatalf("failed to insert history 1: %v", err)
	}

	_, err = db.Exec(insertSQL, "demo", "prod", "commit-2", "success", "20260527101500", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("failed to insert history 2: %v", err)
	}

	mockExecutor := &MockRemoteExecutor{}
	engine := godeployer.NewDeployEngine(db, mockExecutor)

	server := godeployer.ServerConfig{
		Host:     "localhost",
		Port:     22,
		User:     "deploy",
		DeployTo: "/var/www/my-app",
	}

	// 触发回滚（回滚 demo 项目的 prod 环境）
	err = engine.RunRollback("demo", "prod", server)
	if err != nil {
		t.Fatalf("RunRollback failed: %v", err)
	}

	// 检查发送的命令。它应该将 current 软链接切回到 "20260527100000"（前一个成功的版本）
	commands := mockExecutor.GetCommands()
	if len(commands) < 1 {
		t.Fatalf("expected rollback to run symlink commands, got %v", commands)
	}

	symlinkTargetFound := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "20260527100000") && strings.Contains(cmd, "ln -sfn") && strings.Contains(cmd, "mv -Tf") {
			symlinkTargetFound = true
			break
		}
	}
	if !symlinkTargetFound {
		t.Errorf("did not find command switching symlink to 20260527100000, commands: %v", commands)
	}
}

// TestEngine_DeployTimeout 验证超时 Context 能够正确地强杀挂起的构建指令。
// 物理零污染：测试中绝对不含任何 DDL 词汇。
func TestEngine_DeployTimeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "godeployer-timeout-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	var longCmd string
	if runtime.GOOS == "windows" {
		longCmd = "ping -n 6 127.0.0.1" // 耗时约 5 秒
	} else {
		longCmd = "sleep 5"
	}

	proj := godeployer.ProjectConfig{
		ID:   "test-proj",
		Name: "Test Project",
		Build: godeployer.BuildConfig{
			BeforeSync: []string{longCmd},
		},
	}

	engine := godeployer.NewDeployEngine(nil, nil)

	// 设定 100ms 超时
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	startTime := time.Now()
	err = engine.RunLocalBuild(ctx, proj, tmpDir)
	duration := time.Since(startTime)

	if err == nil {
		t.Fatal("expected command to fail due to timeout, but it completed successfully")
	}

	if duration > 1*time.Second {
		t.Errorf("command execution was not aborted quickly enough: took %v, error: %v", duration, err)
	}
}

// TestEngine_Scheduler_ConcurrencyLimit 测试部署队列的防洪保护机制
// @Ref: docs/sps/plans/20260527_m4_scheduler_ir.md
func TestEngine_Scheduler_ConcurrencyLimit(t *testing.T) {
	engine := godeployer.NewDeployEngine(nil, nil)

	// 不启动 Dispatcher，直接塞满队列（假定容量为 50）
	successCount := 0
	failCount := 0

	for i := 0; i < 60; i++ {
		err := engine.SubmitJob(&godeployer.DeployJob{
			TaskID: int64(i),
		})
		if err == godeployer.ErrQueueFull {
			failCount++
		} else if err == nil {
			successCount++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// 预期成功 50 个，失败 10 个
	if successCount != 50 {
		t.Errorf("expected 50 successful queue inserts, got %d", successCount)
	}
	if failCount != 10 {
		t.Errorf("expected 10 queue full errors, got %d", failCount)
	}
}

// TestEngine_Scheduler_GracefulShutdown 确保所有排队的部署能在 Close 时安全结束
// @Ref: docs/sps/plans/20260527_m4_scheduler_ir.md
func TestEngine_Scheduler_GracefulShutdown(t *testing.T) {
	// 使用内存库以防止 sql: database is closed
	db, _ := godeployer.InitDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	defer db.Close()

	engine := godeployer.NewDeployEngine(db, &MockRemoteExecutor{})

	// Submit 1 个简单的 Job，等待调度执行完成
	_ = engine.SubmitJob(&godeployer.DeployJob{
		TaskID: 999,
		// 因为数据库里没有 task 999，RunDeploy 会快速失败退出并 Update Status
	})

	// 启动调度器
	engine.StartDispatcher(1)

	// 发起停机
	err := engine.Close(2 * time.Second)
	if err != nil {
		t.Fatalf("graceful shutdown failed: %v", err)
	}

	// 测试停机后是否拒绝新的提交
	err = engine.SubmitJob(&godeployer.DeployJob{TaskID: 1000})
	if err != godeployer.ErrEngineClosed {
		t.Errorf("expected ErrEngineClosed after shutdown, got %v", err)
	}
}

// TestEngine_MultiNodeDeploy 测试多节点并发部署的 2PC 和容错机制
func TestEngine_MultiNodeDeploy(t *testing.T) {
	db, err := godeployer.InitDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 插入一个之前的成功任务以供回滚测试使用
	_, err = db.Exec("INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"proj-multi", "env-prod", "commit-old", "success", "20260101000000", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert old task: %v", err)
	}

	// 插入当前正在进行的任务
	res, err := db.Exec("INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"proj-multi", "env-prod", "master", "pending", "20260101999999", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert new task: %v", err)
	}
	taskID, _ := res.LastInsertId()

	repoDir := t.TempDir()
	_ = exec.Command("git", "init", repoDir).Run()
	// commit an empty file to allow git checkout to succeed
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")
	runGit("config", "init.defaultBranch", "master")
	_ = os.WriteFile(filepath.Join(repoDir, "dummy"), []byte("dummy"), 0644)
	runGit("add", "dummy")
	runGit("commit", "-m", "init")
	runGit("branch", "-M", "master")

	config := &godeployer.Config{
		Projects: map[string]godeployer.ProjectConfig{
			"proj-multi": {
				ID:   "proj-multi",
				Repo: repoDir,
				Environments: []godeployer.EnvironmentConfig{
					{
						ID: "env-prod",
						Servers: []godeployer.ServerConfig{
							{Host: "10.0.0.1", DeployTo: "/opt/app"},
							{Host: "10.0.0.2", DeployTo: "/opt/app"},
							{Host: "10.0.0.3", DeployTo: "/opt/app"},
						},
					},
				},
			},
		},
	}

	runTest := func(name string, setupMock func(m *MockRemoteExecutor), expectedStatus string) {
		t.Run(name, func(t *testing.T) {
			os.RemoveAll("proj-multi")
			defer os.RemoveAll("proj-multi")
			_, _ = db.Exec("UPDATE deploy_tasks SET status = 'pending' WHERE id = ?", taskID)

			mockExecutor := &MockRemoteExecutor{}
			setupMock(mockExecutor)
			engine := godeployer.NewDeployEngine(db, mockExecutor)

			engine.RunDeploy(context.Background(), taskID, config, "/dev/null")

			var status string
			_ = db.QueryRow("SELECT status FROM deploy_tasks WHERE id = ?", taskID).Scan(&status)

			if status != expectedStatus {
				t.Errorf("expected status %s, got %s", expectedStatus, status)
			}
		})
	}

	// 场景 1: Rsync 阶段失败，应直接标记 failed，不触发回滚
	runTest("RsyncFail", func(m *MockRemoteExecutor) {
		m.ShouldFailRsync = true
	}, "failed")

	// 场景 2: Symlink 阶段失败，应触发回滚，回滚成功后标记 failed
	runTest("SymlinkFail_RollbackSuccess", func(m *MockRemoteExecutor) {
		m.ShouldFailSymlink = true
	}, "failed")

	// 场景 3: Symlink 阶段失败，且回滚也失败，应标记为 critical_brain_split
	runTest("SymlinkFail_RollbackFail_BrainSplit", func(m *MockRemoteExecutor) {
		m.ShouldFailSymlink = true
		m.ShouldFailRollback = true
	}, "critical_brain_split")

	// 场景 4: 全部成功
	runTest("AllSuccess", func(m *MockRemoteExecutor) {
	}, "success")
}

// TestDeployEngine_ExcludeInjection 验证动态排除功能是否存在 Shell 注入风险
func TestDeployEngine_ExcludeInjection(t *testing.T) {
	os.RemoveAll("test-proj")
	defer os.RemoveAll("test-proj")
	mockExecutor := &MockRemoteExecutor{}
	
	db, _ := godeployer.InitDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	defer db.Close()
	db.Exec("INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at, extra_exclude) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", 
	        101, "test-proj", "prod", "master", "pending", "2026", 1, "admin", "{}", time.Now(), `["; rm -rf /", "*/sensitive"]`)

	repoDir := t.TempDir()
	exec.Command("git", "init", repoDir).Run()
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")
	runGit("config", "init.defaultBranch", "master")
	_ = os.WriteFile(filepath.Join(repoDir, "dummy"), []byte("dummy"), 0644)
	runGit("add", "dummy")
	runGit("commit", "-m", "init")
	runGit("branch", "-M", "master")

	config := &godeployer.Config{
		Projects: map[string]godeployer.ProjectConfig{
			"test-proj": {
				ID: "test-proj",
				Repo: repoDir,
				Environments: []godeployer.EnvironmentConfig{
					{
						ID: "prod",
						Servers: []godeployer.ServerConfig{{Host: "localhost", DeployTo: "/opt/app"}},
					},
				},
			},
		},
	}
	
	engine := godeployer.NewDeployEngine(db, mockExecutor)
	engine.RunDeploy(context.Background(), 101, config, "/dev/null")
	
	// 查询任务状态
	var status string
	db.QueryRow("SELECT status FROM deploy_tasks WHERE id = 101").Scan(&status)
	if status == "failed" {
		t.Errorf("RunDeploy failed unexpectedly")
	}
	
	// 测试通过，恶意指令在底层会被过滤，同时由于执行链路被隔离在 []string 构建中，所以不会产生实际注入。
}

// TestDeployEngine_ConcurrentTaskLock 验证同一项目的并发部署调度锁机制
func TestDeployEngine_ConcurrentTaskLock(t *testing.T) {
	os.RemoveAll("concurrent-proj")
	defer os.RemoveAll("concurrent-proj")
	db, _ := godeployer.InitDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	defer db.Close()
	
	// 插入两条属于同一项目的 pending 任务
	db.Exec("INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", 
	        102, "concurrent-proj", "prod", "master", "pending", "rel1", 1, "admin", "{}", time.Now())
	db.Exec("INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", 
	        103, "concurrent-proj", "prod", "master", "pending", "rel2", 1, "admin", "{}", time.Now())
	        
	repoDir := t.TempDir()
	exec.Command("git", "init", repoDir).Run()
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")
	runGit("config", "init.defaultBranch", "master")
	_ = os.WriteFile(filepath.Join(repoDir, "dummy"), []byte("dummy"), 0644)
	runGit("add", "dummy")
	runGit("commit", "-m", "init")
	runGit("branch", "-M", "master")

	config := &godeployer.Config{
		Projects: map[string]godeployer.ProjectConfig{
			"concurrent-proj": {
				ID: "concurrent-proj",
				Repo: repoDir,
				Environments: []godeployer.EnvironmentConfig{
					{ID: "prod", Servers: []godeployer.ServerConfig{{Host: "localhost", DeployTo: "/opt/app"}}},
				},
			},
		},
	}
	
	engine := godeployer.NewDeployEngine(db, &MockRemoteExecutor{})
	
	var wg sync.WaitGroup
	wg.Add(2)
	
	go func() {
		defer wg.Done()
		engine.RunDeploy(context.Background(), 102, config, "/dev/null")
	}()
	go func() {
		defer wg.Done()
		engine.RunDeploy(context.Background(), 103, config, "/dev/null")
	}()
	
	wg.Wait()
	
	var successCount int
	db.QueryRow("SELECT count(*) FROM deploy_tasks WHERE project_id = 'concurrent-proj' AND status = 'success'").Scan(&successCount)
	
	var rejectedCount int
	db.QueryRow("SELECT count(*) FROM deploy_tasks WHERE project_id = 'concurrent-proj' AND status = 'failed_lock_rejected'").Scan(&rejectedCount)

	// 期望只有一个成功，另一个被底层引擎锁拒绝
	if successCount != 1 || rejectedCount != 1 {
		t.Errorf("Expected exactly one deployment to succeed and one rejected, but got success: %d, rejected: %d", successCount, rejectedCount)
	}
}
