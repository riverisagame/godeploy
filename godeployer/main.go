// ============================================================
// 文件：main.go
// 作用：🚀 GoDeploy 项目的主入口文件！
// 这是整个部署系统的"大门口"，所有程序都从这里开始运行。
// 就好比一栋大楼的正门——所有人进出都要经过这里。
//
// 这个文件负责：
// 1. 加载配置（告诉程序怎么跑）
// 2. 连接数据库（准备保存数据的地方）
// 3. 启动 HTTP 服务（让用户可以通过网页访问）
// 4. 启动部署引擎（准备好自动部署的能力）
// 5. 优雅退出（程序关闭时"先打扫卫生再走"）
//
// 给初二小白的解释：
// - package: 就是"包"，把相关的代码打包在一起
// - import: 就是"引入"，把别人写好的功能拿来用
// - func: 函数，就是一段有名字的代码块，可以反复使用
//
// Go 语言的特点：简单、快速、天生支持并发（同时做多件事）！
// ============================================================

package godeployer

// 📦 import 的作用就像"点外卖"——我们从不同的"店"（包）里点我们需要的东西
import (
	// ---------- 项目内部的包（我们自己写的代码） ----------
	"deploy/godeployer/application"       // 🎯 应用层：存放"部署引擎"这种核心业务逻辑
	"deploy/godeployer/domain"            // 📋 领域层：存放核心概念（配置、任务、状态等）
	"deploy/godeployer/infrastructure/notifier" // 📢 通知器：当有事情发生时，向所有人广播消息
	"deploy/godeployer/infrastructure/ssh"      // 🔒 SSH 工具：通过 SSH 协议控制远程服务器
	"deploy/godeployer/infrastructure/db"       // 💾 数据库：存取任务记录、用户信息
	"deploy/godeployer/interfaces/api"          // 🌐 API 接口：处理来自网页的请求
	"deploy/godeployer/infrastructure/config"   // ⚙️ 配置加载器：读取 YAML 配置文件

	// ---------- Go 语言标准库（Go 自带的官方工具包） ----------
	"context"     // 📦 上下文：用来控制超时、取消任务等
	"database/sql" // 🗄️ SQL 数据库的通用接口
	"embed"       // 📎 嵌入：可以把前端网页"嵌入"到 Go 程序中
	"flag"        // 🚩 命令行参数：比如 `--config=xxx.yaml`
	"fmt"         // ✏️ 格式化输出：打印文字到屏幕上
	"io/fs"       // 📁 文件系统：操作文件和目录
	"log"         // 📝 日志：在屏幕上打印程序运行信息
	"net/http"    // 🌍 HTTP 服务：让程序可以被网页浏览器访问
	"os"          // 💻 操作系统相关：获取系统信号、操作文件等
	"os/signal"   // 🚦 信号处理：捕获 Ctrl+C、系统关机等信号
	"strings"     // 📏 字符串工具：查找、替换、分割文本
	"syscall"     // ⚙️ 系统调用：与操作系统内核直接交互
	"time"        // ⏰ 时间工具：处理时间、设置超时等

	// ---------- 第三方库（别人写好的轮子） ----------
	"github.com/gin-gonic/gin" // 🚄 Gin 框架：超快的 Go Web 框架，像搭建积木一样搭建网站后端
)

// 🎯 //go:embed 是 Go 的一个"编译指令"
// 它的作用：在编译程序时，把 web/dist/ 这个文件夹里的所有文件
// 都打包到最终生成的二进制程序中！
// 这样我们只需要一个 .exe 文件就能运行整个系统，不需要额外带前端文件。
//
// 想象一下：你把乐高积木全部装进一个盒子里带走，而不是拿盒子+散落的积木。
//
//go:embed dist
var embeddedFiles embed.FS // embeddedFiles 是一个"虚拟文件系统"，里面装着前端网页

// GetEmbeddedAsset 从嵌入的前端文件系统中读取指定文件。
// 比如前端需要一个叫 logo.png 的图片，这个函数就去"虚拟文件箱"里找出来。
//
// 参数 path：文件的路径，比如 "index.html" 或 "assets/style.css"
// 返回值1：文件的内容（一堆字节）
// 返回值2：如果有错误就告诉你
func GetEmbeddedAsset(path string) ([]byte, error) {
	// embeddedFiles.ReadFile(path) 就是在"虚拟文件箱"里翻找
	return embeddedFiles.ReadFile(path)
}

// BootstrapApp 是"引导启动"函数，就像做菜前先准备好所有的食材和工具！
//
// 它负责两件大事：
// 1. 加载配置文件（读 YAML 文件，告诉程序各种设置）
// 2. 初始化数据库（连接 SQLite 数据库，准备存数据）
//
// 参数 configPath：配置文件（.yaml）的路径
// 返回值1：配置信息（所有的设置项）
// 返回值2：数据库连接（操作数据库的"手柄"）
// 返回值3：任务仓库（操作部署任务的工具）
// 返回值4：如果出错了，告诉你什么错
func BootstrapApp(configPath string) (*domain.Config, *sql.DB, domain.TaskRepository, error) {
	// --- 第 1 步：加载配置 ---
	// config.LoadConfig 会读取 YAML 文件，并解析成程序能看懂的结构
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// ❌ 如果配置加载失败，直接返回错误
		// %w 是 Go 的"包装错误"语法，就像给错误外面再包一层说明
		return nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// --- 第 2 步：连接数据库 ---
	// db.InitGORM 会打开 SQLite 数据库文件，并自动创建需要的表
	// 第一个参数 "sqlite" 表示使用 SQLite 数据库
	// 第二个参数是数据库文件的存放路径
	gormDB, err := db.InitGORM("sqlite", cfg.Global.SQLitePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	// 从 GORM 的数据库对象中获取底层的原生 sql.DB 对象
	// 就像从"自动挡汽车"里拿出"手动挡操作杆"~
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// --- 第 3 步：创建任务仓库 ---
	// 任务仓库 = 专门用来操作"部署任务"这个表的工具
	// 比如：创建一个新任务、查询任务状态、更新任务进度
	taskRepo := db.NewTaskRepository(gormDB)

	// ✅ 所有准备都做好了，把工具返回给调用者
	return cfg, sqlDB, taskRepo, nil
}

// SetupStaticEmbed 设置前端的静态文件服务。
//
// 什么是"静态文件"？就是那些不会变的文件——HTML、CSS、JavaScript、图片等。
// 我们的 Vue 前端编译后会变成一堆静态文件，嵌入在 Go 程序里。
// 这个函数的作用就是：当用户在浏览器访问时，把前端页面"喂"给浏览器。
//
// 它还做了一个特别重要的事：SPA Fallback（单页面应用回退）！
// Vue 路由可能长这样：/projects/123，但服务器上没有这个文件，
// 这时就需要把请求指向 index.html，让 Vue 自己处理路由。
func SetupStaticEmbed(r *gin.Engine) {
	// 从整个嵌入文件系统中，取出 "dist" 这个子文件夹
	// fs.Sub 就像从一个大文件柜里抽出一个抽屉
	distFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		// 如果取不出来（比如前端还没编译），程序就直接崩溃报错
		log.Fatalf("failed to create dist sub FS: %v", err)
	}

	// 创建一个标准的 HTTP 文件服务器
	// 这个服务器知道怎么把文件内容发给浏览器
	fileServer := http.FileServer(http.FS(distFS))

	// r.NoRoute 是 Gin 框架的"兜底"处理：
	// 当用户访问的地址没有匹配到任何 API 路由时，就由这里的代码处理
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path // 用户访问的路径，比如 /about 或 /api/login

		// --- 规则 1：API 请求不归我们管 ---
		// 如果路径以 /api 开头，说明是在请求数据接口，不是要页面
		if strings.HasPrefix(path, "/api") {
			// 返回 404 错误，告诉前端"这个接口不存在"
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return // ✅ 处理完毕，退出
		}

		// --- 规则 2：看看用户要的是不是某个存在的文件 ---
		filePath := strings.TrimPrefix(path, "/") // 去掉前面的斜杠，比如 /logo.png → logo.png
		if filePath == "" {
			// 如果用户访问的是根路径 /，就默认返回 index.html
			filePath = "index.html"
		}

		// 尝试在"虚拟文件箱"里找这个文件
		f, err := distFS.Open(filePath)
		if err != nil {
			// ❌ 文件不存在！
			// 这通常是 Vue 的路由（比如 /projects/123）
			// 对于 SPA（单页面应用），所有的前端路由都要返回 index.html
			// 让 Vue 自己根据 URL 显示对应的页面内容
			indexData, err := embeddedFiles.ReadFile("dist/index.html")
			if err != nil {
				// 如果连 index.html 都没有，那就真的出大问题了
				c.String(http.StatusInternalServerError, "Internal Server Error")
				return
			}
			// ✅ 返回 index.html，告诉浏览器"你来解析这个页面"
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
			return
		}
		f.Close() // 用完文件要关上，就像看完书要合上

		// ✅ 文件存在，让文件服务器正常返回这个文件
		// 比如用户访问 /favicon.ico，就直接返回图标文件
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// StartServer 是"幕后大总管"，负责整个程序的启动流程！
// 这个函数从 main.go 中被调用，然后按顺序：
// 1. 读取命令行参数（用户启动时给的选项）
// 2. 引导启动（加载配置 + 初始化数据库）
// 3. 初始化消息总线（方便不同模块之间"喊话"）
// 4. 创建部署引擎（准备执行部署任务）
// 5. 搭建路由 + 挂载前端页面
// 6. 启动 HTTP 服务器（让用户能通过浏览器访问）
// 7. 等待退出信号（等用户按 Ctrl+C）
// 8. 优雅关闭（先让手里的活干完再下班）
func StartServer() {
	// --- 第 1 步：读取命令行参数 ---
	// flag.String 定义了一个命令行选项 --config
	// 默认值是 "config.yaml"
	// 用户可以用：./godeployer --config=myconfig.yaml 来指定配置文件
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse() // 🚩 解析用户输入的命令行参数

	log.Printf("Starting GoDeployer using config: %s", *configPath)

	// --- 第 2 步：引导启动（加载配置 + 数据库） ---
	config, sqlDB, taskRepo, err := BootstrapApp(*configPath)
	if err != nil {
		// ❌ 引导失败，直接退出程序
		log.Fatalf("Application bootstrap failed: %v", err)
	}
	// defer 是一个"延迟执行"的关键字
	// 意思就是：不管函数怎么结束，最后都要执行 sqlDB.Close()
	// 就像出门前一定记得锁门一样！
	defer sqlDB.Close()

	// --- 第 3 步：初始化事件通知总线 ---
	// EventBus（事件总线）就像公司里的广播系统
	// 有人完成了部署，广播一下，所有人都知道
	// 这也叫"发布-订阅模式"——有人发布消息，有人订阅消息
	bus := notifier.NewEventBus()
	bus.StartEventConsumer(10) // 📢 启动 10 个"广播员"并发处理消息

	// --- 第 4 步：创建部署引擎 ---
	// NodeAdapter（节点适配器）是操作远程服务器的工具
	// DeploymentService 是部署的核心规则
	// DeployEngine 是具体的执行者，就像一个"施工队长"
	nodeAdapter := ssh.NewNodeAdapter()
	deploySvc := domain.NewDeploymentService(taskRepo)
	engine := application.NewDeployEngine(taskRepo, nodeAdapter, deploySvc)
	engine.StartDispatcher(3) // 🏭 启动 3 个工人同时处理不同的部署任务

	// --- 第 5 步：创建路由 + 挂载前端 ---
	// Gin 的路由就像一张地图：/api/login → 登录处理函数
	r := api.SetupRoutes(config, sqlDB, taskRepo, engine)

	// 把前端静态文件挂载到路由上
	SetupStaticEmbed(r)

	// --- 第 6 步：启动 HTTP 服务器 ---
	// 构造服务器的地址，比如 :8080
	addr := fmt.Sprintf(":%d", config.Global.ServerPort)
	log.Printf("GoDeployer web console is running on http://localhost%s", addr)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    addr,      // 监听地址（比如 :8080）
		Handler: r,         // 请求处理（用刚才的路由地图）
	}

	// go func() 启动一个"协程"——相当于开启了一条新的工作线
	// 这样服务器在后台监听，主线程可以继续做别的事
	go func() {
		// srv.ListenAndServe() 开始监听端口，等待用户访问
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// --- 第 7 步：等待退出信号 ---
	// 这行代码创建一个"通道"（channel），可以理解为一根管子
	// 当用户按下 Ctrl+C，操作系统会发送一个信号，通过管子传过来
	quit := make(chan os.Signal, 1)

	// 告诉操作系统："我要监听 SIGINT（Ctrl+C）和 SIGTERM（系统关机）信号"
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 【阻塞等待】这行代码会停在这里，直到有人按 Ctrl+C
	// 就像在大厅里等人喊你回家吃饭
	<-quit
	log.Println("Shutting down server...")

	// --- 第 8 步：优雅关闭 ---
	// 创建一个"带超时的上下文"，最多等 5 秒
	// 就像跟客户说："给我 5 秒收个尾"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 告诉 HTTP 服务器："停止接新客人，但把现在的客人服务完"
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// 关闭事件总线，把最后几条广播发完
	log.Println("Flushing EventBus...")
	if err := bus.Close(5 * time.Second); err != nil {
		log.Printf("EventBus close error: %v", err)
	}

	log.Println("Server exiting")

	// 关闭部署引擎，给正在进行的部署最多 30 秒完成
	// 就像告诉施工队长："等工人干完手头的活再收工"
	log.Println("Waiting for active deployments to finish...")
	if err := engine.Close(30 * time.Second); err != nil {
		log.Printf("DeployEngine close error: %v", err)
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: package 关键字是做什么的？
//    A: 把相关的 Go 文件打包在一起，方便管理和使用~
//
// 2. Q: import 是什么？
//    A: 就像手机装 App——从别的地方拿现成的功能来用！
//
// 3. Q: //go:embed 指令的作用？
//    A: 编译时把前端网页文件"塞进"程序里，这样只需要一个文件就能跑~
//
// 4. Q: defer 关键字有什么作用？
//    A: 延迟执行，函数结束前一定会执行！就像"不管怎样，回家前都要锁门"~
//
// 5. Q: 什么是 goroutine（go func()）？
//    A: 轻量级的"线程"，可以同时做多件事，比如后台监听、前台处理请求~
//
// 中级（面试常考）：
// 6. Q: SPA（单页面应用）的 Fallback 机制是什么？
//    A: 当用户访问一个前端路由 URL（比如 /projects/123），但服务器上没有这个文件时，
//       把请求降级到 index.html，让 Vue 自己处理路由~
//
// 7. Q: context.WithTimeout 是做什么的？
//    A: 创建一个"超时上下文"，超过指定时间就自动取消，防止程序无限等待~
//
// 8. Q: 通道（channel）是什么？
//    A: goroutine 之间通信的管道！可以发数据和收数据。<-quit 就是从管子收信号~
//
// 高级（架构师级别）：
// 9. Q: 为什么需要优雅关闭（Graceful Shutdown）？
//    A: 防止正在处理的请求被强行中断！先停止接新请求 → 等正在处理的完成 → 再关闭资源
//
// 10. Q: 嵌入前端到 Go 二进制的优缺点？
//     A: 优点：部署简单，一个文件搞定！缺点：前端更新需要重新编译整个程序~
// ============================================================
