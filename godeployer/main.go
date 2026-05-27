package godeployer

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var embeddedFiles embed.FS

// GetEmbeddedAsset 允许读取内嵌的前端静态资源。
func GetEmbeddedAsset(path string) ([]byte, error) {
	return embeddedFiles.ReadFile(path)
}

// BootstrapApp 提供集成化的配置加载与数据库初始化引导。
func BootstrapApp(configPath string) (*Config, *sql.DB, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := InitDB(config.Global.SQLitePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	return config, db, nil
}

// SetupStaticEmbed 挂载前端静态资源并提供 SPA Fallback 机制。
func SetupStaticEmbed(r *gin.Engine) {
	distFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		log.Fatalf("failed to create dist sub FS: %v", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路由交由 API 处理，不做静态 Fallback
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}

		filePath := strings.TrimPrefix(path, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// 检查静态资源是否存在
		f, err := distFS.Open(filePath)
		if err != nil {
			// 文件不存在，Fallback 到 Vue 单页面路由 index.html
			indexData, err := embeddedFiles.ReadFile("dist/index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, "Internal Server Error")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
			return
		}
		f.Close()

		// 资源存在，正常返回
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// StartServer 供命令行调用的后端启动点
func StartServer() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	log.Printf("Starting GoDeployer using config: %s", *configPath)

	config, db, err := BootstrapApp(*configPath)
	if err != nil {
		log.Fatalf("Application bootstrap failed: %v", err)
	}
	defer db.Close()

	// 初始化事件通知总线并异步启动 (启动 10 个 Worker)
	bus := NewEventBus()
	bus.StartEventConsumer(10)

	// 实例化部署引擎并启动 Dispatcher (最大并发 3)
	engine := NewDeployEngine(db, nil)
	engine.StartDispatcher(3)

	// 创建路由
	r := SetupRoutes(config, db, engine)

	// 挂载静态网页
	SetupStaticEmbed(r)

	addr := fmt.Sprintf(":%d", config.Global.ServerPort)
	log.Printf("GoDeployer web console is running on http://localhost%s", addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// 监听中断信号，触发优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 最多等待 5 秒完成现有请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// 优雅关闭 EventBus，最多等待 5 秒让缓冲中事件发送完
	log.Println("Flushing EventBus...")
	if err := bus.Close(5 * time.Second); err != nil {
		log.Printf("EventBus close error: %v", err)
	}

	log.Println("Server exiting")
	
	// 优雅关闭 DeployEngine，允许进行中的部署完成（最多给 30 秒宽限期）
	log.Println("Waiting for active deployments to finish...")
	if err := engine.Close(30 * time.Second); err != nil {
		log.Printf("DeployEngine close error: %v", err)
	}
}
