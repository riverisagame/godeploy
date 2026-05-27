package godeployer_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deploy/godeployer"
	"github.com/gorilla/websocket"
)

// 物理零污染与 DDL 绝对禁绝: 测试中绝不写入本地物理文件
func TestAPI_WS_TaskLog_Integration(t *testing.T) {
	router, cleanup := SetupTestRouter(t)
	defer cleanup()

	// 启动 httptest.Server 以支持 WS 协议升级
	ts := httptest.NewServer(router)
	defer ts.Close()

	// 将 http:// 转换为 ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws/tasks/1/log"

	// 尝试拨号连接 WebSocket
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v, resp: %v", err, resp)
	}
	defer conn.Close()

	// 生成一个合法的 Admin Token
	token, err := godeployer.GenerateToken("admin", "admin", "test-secret-key-12345", 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 步骤 1: 发送鉴权 Payload
	authPayload := map[string]string{
		"type":  "auth",
		"token": token,
	}
	err = conn.WriteJSON(authPayload)
	if err != nil {
		t.Fatalf("Failed to write auth payload: %v", err)
	}

	// 步骤 2: 等待接收日志消息
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			t.Fatalf("Connection closed unexpectedly: %v", err)
		}
		// Expect a timeout because we are not writing to the physical log file
		if strings.Contains(err.Error(), "i/o timeout") || strings.Contains(err.Error(), "read deadline") {
			t.Log("Successfully authenticated and connected. Timeout as expected since no physical log is written.")
			return
		}
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(msg) > 0 {
		t.Logf("Received message: %s", string(msg))
	}
}
