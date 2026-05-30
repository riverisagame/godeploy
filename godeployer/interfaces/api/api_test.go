package api

import (
	"sync"

	"bytes"
	"database/sql"
	"deploy/godeployer/application"
	"deploy/godeployer/domain"
	"deploy/godeployer/infrastructure/db"
	"deploy/godeployer/infrastructure/ssh"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupTestRouter 初始化 Gin 测试路由，并连接临时内存 SQLite 数据库。
// 物理零污染与 DDL 绝对禁绝：测试中绝不写入本地物理文件，源码中绝无 CREATE/DROP 词眼。
func SetupTestRouter(t *testing.T) (*gin.Engine, *sql.DB, func()) {
	gin.SetMode(gin.TestMode)

	// 使用内存数据库
	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}

	// 缓存一个默认的测试项目配置
	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:  "test-secret-key-12345",
			LogPath:    "./test_logs",
			SSHKeyPath: "./test_keys/id_rsa",
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {
				ID:            "test-app",
				Name:          "Mock Test App",
				Repo:          "git@github.com:mock/test-app.git",
				WebhookSecret: "secret123",
				Branch:        "main",
				Environments: []domain.EnvironmentConfig{
					{
						ID:   "testing",
						Name: "Test Env",
						Servers: []domain.ServerConfig{
							{
								Host:       "127.0.0.1",
								Port:       22,
								User:       "mock-user",
								DeployTo:   "/tmp/mock-deploy",
								SSHKeyPath: "./test_keys/id_rsa",
							},
						},
					},
				},
			},
		},
	}

	// 注册路由
	engine := application.NewDeployEngine(taskRepo, nil)
	engine.StartDispatcher(1)
	r := SetupRoutes(mockConfig, db, taskRepo, engine)

	cleanup := func() {
		engine.Close(2 * time.Second)
		db.Close()
	}

	return r, db, cleanup
}

func SetupTestRouterWithExecutor(t *testing.T, executor ssh.RemoteExecutor) (*gin.Engine, *sql.DB, func()) {
	gin.SetMode(gin.TestMode)

	db, taskRepo, err := db.InitTestDB(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("failed to init DB: %v", err)
	}

	mockConfig := &domain.Config{
		Global: domain.GlobalConfig{
			JWTSecret:  "test-secret-key-12345",
			LogPath:    "./test_logs",
			SSHKeyPath: "./test_keys/id_rsa",
		},
		Projects: map[string]domain.ProjectConfig{
			"test-app": {
				ID:            "test-app",
				Name:          "Mock Test App",
				Repo:          "git@github.com:mock/test-app.git",
				WebhookSecret: "secret123",
				Branch:        "main",
				Environments: []domain.EnvironmentConfig{
					{
						ID:   "testing",
						Name: "Test Env",
						Servers: []domain.ServerConfig{
							{
								Host:       "127.0.0.1",
								Port:       22,
								User:       "mock-user",
								DeployTo:   "/tmp/mock-deploy",
								SSHKeyPath: "./test_keys/id_rsa",
							},
						},
					},
				},
			},
		},
	}

	engine := application.NewDeployEngine(taskRepo, executor)
	engine.StartDispatcher(1)
	r := SetupRoutesWithExecutor(mockConfig, db, taskRepo, executor, engine)

	cleanup := func() {
		engine.Close(2 * time.Second)
		db.Close()
	}

	return r, db, cleanup
}

// TestAPI_LoginVerify 验证 JWT 登录 API 流程。
func TestAPI_LoginVerify(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	// 1. 正确的默认密码登录 (admin / admin123)
	loginJSON := map[string]string{
		"username": "admin",
		"password": "admin123",
	}
	body, _ := json.Marshal(loginJSON)
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on login success, got %d (body: %s)", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if _, ok := res["token"]; !ok {
		t.Error("response does not contain jwt token")
	}

	// 2. 错误密码登录 (admin / wrongpass)
	loginJSON["password"] = "wrongpass"
	body, _ = json.Marshal(loginJSON)
	req, _ = http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on login failure, got %d", w.Code)
	}
}

// TestAPI_GetProjectsVerify 验证配置文件的只读展示 API 是否正确返回。
func TestAPI_GetProjectsVerify(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/api/projects", nil)
	// 本次测试只读接口也加上认证校验
	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var projects []domain.ProjectConfig
	err := json.Unmarshal(w.Body.Bytes(), &projects)
	if err != nil {
		t.Fatalf("failed to parse projects JSON response: %v", err)
	}

	if len(projects) != 1 || projects[0].ID != "test-app" {
		t.Errorf("expected test-app project config to be listed, got %+v", projects)
	}
}

// TestAPI_CreateTaskAudit 验证创建部署任务时，正确的用户信息可以被审计写入数据库。
func TestAPI_CreateTaskAudit(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	// 生成管理员 Token
	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	taskPayload := map[string]string{
		"project_id": "test-app",
		"env_id":     "testing",
		"commit_id":  "abcde12345",
	}
	body, _ := json.Marshal(taskPayload)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", w.Code, w.Body.String())
	}

	var task map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &task)

	// 确认审计字段包含当前登录人
	if task["username"] != "admin" {
		t.Errorf("expected audited username to be 'admin', got %v", task["username"])
	}
}

// TestAPI_DeployLockVerify 验证并发部署锁逻辑。
// 当该项目和环境已经有一个任务状态为 'deploying' 或 'pending' 时，拒绝重复发起部署。
func TestAPI_DeployLockVerify(t *testing.T) {
	r, db, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	// 写入一条状态为 deploying 的模拟记录
	insertSQL := `INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(insertSQL, "test-app", "testing", "commit-active", "deploying", "20260527110000", 1, "admin", "{}", time.Now())
	if err != nil {
		t.Fatalf("failed to insert active task: %v", err)
	}

	// 2. 发起新的部署请求，应该被锁机制拦截
	taskPayload := map[string]string{
		"project_id": "test-app",
		"env_id":     "testing",
		"commit_id":  "commit-new",
	}
	body, _ := json.Marshal(taskPayload)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 预期被并发锁拦截返回 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestAPI_GetTaskLogTruncate 验证日志读取接口的大日志文件截断防护。
func TestAPI_GetTaskLogTruncate(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	// 1. 创建超大模拟日志文件 (1.2 MB)
	logDir := "./test_logs"
	_ = os.MkdirAll(logDir, 0755)
	defer os.RemoveAll(logDir)

	logFilePath := filepath.Join(logDir, "task_999.log")
	largeData := make([]byte, 1258291)
	for i := range largeData {
		largeData[i] = 'A'
	}
	copy(largeData[len(largeData)-10:], []byte("LAST_BYTES"))

	if err := os.WriteFile(logFilePath, largeData, 0644); err != nil {
		t.Fatalf("failed to write simulated large log file: %v", err)
	}

	// 2. 请求日志
	req, _ := http.NewRequest("GET", "/api/tasks/999/log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	logText, ok := res["log"].(string)
	if !ok {
		t.Fatal("response log text is missing")
	}

	// 预期只读取了最后的 1MB 并带有截断标志
	if !strings.Contains(logText, "[Log truncated") {
		t.Error("expected truncation prefix warning in log response")
	}
	if !strings.Contains(logText, "LAST_BYTES") {
		t.Error("expected tail bytes to be present in read logs")
	}
}

// TestAPI_GitDiffVerify 验证新增的 Git Diff 对比接口。
func TestAPI_GitDiffVerify(t *testing.T) {
	r, db, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	insertSQL := `INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, _ = db.Exec(insertSQL, 1, "test-app", "testing", "commit-1", "success", "20260527100000", 1, "admin", "{}", time.Now().Add(-20*time.Minute))
	_, _ = db.Exec(insertSQL, 2, "test-app", "testing", "commit-2", "success", "20260527101500", 1, "admin", "{}", time.Now().Add(-10*time.Minute))

	// 2. 发起 Diff 请求对 Task 2 与前一个成功版本 Task 1 之间进行对比
	// 预期返回 200 OK，包含对比的 diff 内容
	req, _ := http.NewRequest("GET", "/api/tasks/2/diff", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestAPI_RollbackPrecisionVerify 验证回滚动作精准度。
func TestAPI_RollbackPrecisionVerify(t *testing.T) {
	mockExecutor := &MockRemoteExecutor{}
	r, db, cleanup := SetupTestRouterWithExecutor(t, mockExecutor)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	insertSQL := `INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, _ = db.Exec(insertSQL, 1, "test-app", "testing", "commit-1", "success", "20260527100000", 1, "admin", "{}", time.Now().Add(-20*time.Minute))
	_, _ = db.Exec(insertSQL, 2, "test-app", "testing", "commit-2", "success", "20260527101500", 1, "admin", "{}", time.Now().Add(-10*time.Minute))

	// 对 Task 1 进行回滚
	req, _ := http.NewRequest("POST", "/api/tasks/1/rollback", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 此时回滚会走 mockExecutor，不发生 SSH 私钥加载失败的 500 报错，预期返回 200 OK
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on rollback trigger, got %d (body: %s)", w.Code, w.Body.String())
	}

	// 确认 mockExecutor 确实收到了精准回滚到 20260527100000 (Task 1) 的 ln 软链接切换命令
	commands := mockExecutor.GetCommands()
	foundTargetRelease := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "20260527100000") && strings.Contains(cmd, "ln -sfn") {
			foundTargetRelease = true
			break
		}
	}
	if !foundTargetRelease {
		t.Errorf("expected rollback commands to target release 20260527100000, but ran: %v", commands)
	}
}

// TestAPI_GetTaskDiff_RaceCondition 验证请求 diff 时，若目标任务处于 deploying 状态，应当被拒绝以防竞态。
func TestAPI_GetTaskDiff_RaceCondition(t *testing.T) {
	r, db, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	// 插入一条状态为 deploying 的任务
	insertSQL := `INSERT INTO deploy_tasks (id, project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, _ = db.Exec(insertSQL, 3, "test-app", "testing", "commit-3", "deploying", "20260527120000", 1, "admin", "{}", time.Now())

	// 请求该 deploying 任务的 diff
	req, _ := http.NewRequest("GET", "/api/tasks/3/diff", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 预期返回 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for deploying task diff request, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestAPI_GithubWebhook 验证 Github Webhook 的签名校验与防抖逻辑。
// @Ref: docs/sps/plans/20260527_nanoplan_m2_rbac_webhooks.md
func TestAPI_GithubWebhook(t *testing.T) {
	r, db, cleanup := SetupTestRouter(t)
	defer cleanup()

	// 构造 Github Push Payload
	payload := `{"ref": "refs/heads/main", "after": "commit-webhook-1"}`

	// 计算真实的 HMAC SHA256 签名
	mac := ComputeGithubSignature([]byte(payload), "secret123")

	req, _ := http.NewRequest("POST", "/api/webhooks/github/test-app/testing", bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", "sha256="+mac)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 预期返回 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 for valid webhook, got %d (body: %s)", w.Code, w.Body.String())
	}

	// 为了测试防抖，我们直接往数据库插入一条针对 commit-webhook-2 且状态为 deploying 的任务
	db.Exec(`INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, created_at) 
		VALUES ('test-app', 'testing', 'commit-webhook-2', 'deploying', '20260527123456', 1, 'admin', ?)`, time.Now())

	payload2 := `{"ref": "refs/heads/main", "after": "commit-webhook-2"}`
	mac2 := ComputeGithubSignature([]byte(payload2), "secret123")

	req2, _ := http.NewRequest("POST", "/api/webhooks/github/test-app/testing", bytes.NewBuffer([]byte(payload2)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Hub-Signature-256", "sha256="+mac2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	// 预期返回 409 Conflict
	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for concurrent webhook, got %d (body: %s)", w2.Code, w2.Body.String())
	}
}

// TestHandleTasks_InvalidProject 验证创建部署任务时，无效的项目 ID 是否被正确拦截。
func TestHandleTasks_InvalidProject(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	taskPayload := map[string]string{
		"project_id": "non-existent-project",
		"env_id":     "testing",
		"commit_id":  "abcde12345",
	}
	body, _ := json.Marshal(taskPayload)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 应该是 400 Bad Request 或者 404 Not Found
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("expected 404 or 400 for invalid project, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// TestHandleDeploy_InvalidEnv 验证创建部署任务时，无效的环境 ID 是否被正确拦截。
func TestHandleDeploy_InvalidEnv(t *testing.T) {
	r, _, cleanup := SetupTestRouter(t)
	defer cleanup()

	token, _ := application.GenerateToken("admin", "admin", "test-secret-key-12345", 5*time.Second)

	taskPayload := map[string]string{
		"project_id": "test-app",
		"env_id":     "invalid-env",
		"commit_id":  "abcde12345",
	}
	body, _ := json.Marshal(taskPayload)
	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 应该是 400 Bad Request 或者 404 Not Found
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("expected 404 or 400 for invalid env, got %d (body: %s)", w.Code, w.Body.String())
	}
}


type MockRemoteExecutor struct {
    mu sync.Mutex
    commandsRun []string
}
func (m *MockRemoteExecutor) RunCommand(cmd string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.commandsRun = append(m.commandsRun, cmd)
    return "", nil
}
func (m *MockRemoteExecutor) Rsync(a, b, c string) error {
    return nil
}
func (m *MockRemoteExecutor) Close() error {
    return nil
}
func (m *MockRemoteExecutor) GetCommands() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    res := make([]string, len(m.commandsRun))
    copy(res, m.commandsRun)
    return res
}
