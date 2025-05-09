package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// 导入公共模块
	sharedCore "github.com/Xushengqwer/go-common/core"
	sharedTracing "github.com/Xushengqwer/go-common/core/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp" // OTel HTTP instrumentation
	"go.uber.org/zap"

	// 导入项目包
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	_ "github.com/Xushengqwer/user_hub/docs" // Swagger 文档，匿名导入以执行其 init()
	"github.com/Xushengqwer/user_hub/initialization"
	"github.com/Xushengqwer/user_hub/router"
)

// @title User Hub API
// @version 1.0
// @description 用户中心服务 API 文档
// @host localhost:8080 // 根据实际部署调整
// @BasePath /api/v1
// @schemes http https
func main() {
	// --- 配置和基础设置 ---
	var configFile string
	// 从命令行参数读取配置文件路径，默认为 "config/config.development.yaml"
	flag.StringVar(&configFile, "config", "config/config.development.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.UserHubConfig
	// 使用公共模块加载配置，cfg 需要是指针
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("FATAL: 加载配置失败 (%s): %v", configFile, err)
	}
	// log.Printf("DEBUG: Loaded config: %+v\n", cfg) // 避免在生产环境打印敏感信息

	// 2. 初始化 Logger
	logger, loggerErr := sharedCore.NewZapLogger(cfg.ZapConfig)
	if loggerErr != nil {
		log.Fatalf("FATAL: 初始化 ZapLogger 失败: %v", loggerErr)
	}
	// 使用 defer 确保在 main 函数退出时同步日志缓冲区
	defer func() {
		logger.Info("正在同步日志...")
		if err := logger.Logger().Sync(); err != nil {
			// Sync 失败通常不影响程序运行，记录警告即可
			log.Printf("WARN: ZapLogger Sync 失败: %v\n", err)
		}
	}()
	logger.Info("Logger 初始化成功")

	// 3. 初始化 TracerProvider (如果启用)
	var tracerShutdown func(context.Context) error = func(ctx context.Context) error { return nil } // 默认为空操作
	if cfg.TracerConfig.Enabled {
		var err error
		// 初始化 TracerProvider，传入服务名、版本和配置
		tracerShutdown, err = sharedTracing.InitTracerProvider(
			constants.ServiceName,    // 服务名常量
			constants.ServiceVersion, // 服务版本常量
			cfg.TracerConfig,         // 追踪配置
		)
		if err != nil {
			logger.Fatal("初始化 TracerProvider 失败", zap.Error(err))
		}
		// 使用 defer 确保追踪系统在程序退出时优雅关闭
		defer func() {
			// 创建一个带超时的上下文用于关闭操作
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			logger.Info("正在关闭 TracerProvider...")
			if err := tracerShutdown(ctx); err != nil {
				logger.Error("关闭 TracerProvider 失败", zap.Error(err))
			} else {
				logger.Info("TracerProvider 已成功关闭")
			}
		}()
		logger.Info("分布式追踪已初始化")
		// 初始化 OTel HTTP Transport (即使暂时不用，初始化也无害)
		_ = otelhttp.NewTransport(http.DefaultTransport)
		logger.Debug("OTel HTTP Transport 初始化完成 (暂未使用)")
	} else {
		logger.Info("分布式追踪已禁用")
	}

	// --- 依赖注入和应用设置 ---

	// 4. 初始化基础依赖 (数据库, Redis, JWT, 外部客户端等)
	//    调用 initialization 包中的 SetupDependencies 函数
	appDeps, err := initialization.SetupDependencies(&cfg, logger)
	if err != nil {
		// 如果基础依赖初始化失败，则无法继续运行
		logger.Fatal("初始化基础依赖失败", zap.Error(err))
	}
	logger.Info("基础依赖初始化成功")

	// 5. 初始化服务层实例
	//    调用 initialization 包中的 SetupServices 函数，传入基础依赖
	appServices := initialization.SetupServices(appDeps)
	logger.Info("服务层初始化成功")

	// 6. 设置路由和中间件
	//    调用 router 包中的 SetupRouter 函数，传入所需依赖
	setupRouter := router.SetupRouter(
		logger,           // 日志记录器
		&cfg,             // 传入完整配置，以便 SetupRouter 获取所需部分
		appDeps.JwtToken, // JWT 工具
		appServices,      // 所有服务实例
	)
	logger.Info("Gin 路由器设置完成")

	// --- 启动 HTTP 服务器 ---

	// 7. 配置并启动 HTTP 服务器
	serverAddress := fmt.Sprintf(":%s", cfg.ServerConfig.Port) // 从配置构建监听地址
	srv := &http.Server{
		Addr: serverAddress,
		// Handler: setupRouter, // Gin 引擎本身就是一个 http.Handler
		// 添加 OTel HTTP 中间件来包裹 Gin 引擎，以便追踪所有 HTTP 请求
		Handler: otelhttp.NewHandler(setupRouter, "HTTPServer"), // "HTTPServer" 是操作名
		// 可以设置读写超时等参数
		// ReadTimeout:  5 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  120 * time.Second,
	}

	// 8. 启动服务器 (使用 goroutine，以便不阻塞后续的优雅关停逻辑)
	go func() {
		logger.Info("HTTP 服务器开始监听", zap.String("address", serverAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// ListenAndServe 在正常关闭时会返回 ErrServerClosed，不应视为 Fatal 错误
			logger.Fatal("HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// --- 优雅关停处理 ---

	// 9. 等待中断信号以实现优雅关停
	quit := make(chan os.Signal, 1) // 创建一个接收信号的 channel
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill 命令默认发送) 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	recSignal := <-quit // 阻塞，直到接收到上述信号之一
	logger.Info("接收到关停信号", zap.String("signal", recSignal.String()))

	// 10. 执行优雅关停
	// 创建一个带超时的上下文，用于服务器关闭操作
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second) // 例如，10秒超时
	defer cancelShutdown()

	logger.Info("开始优雅关停 HTTP 服务器...")
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("HTTP 服务器优雅关停失败", zap.Error(err))
	} else {
		logger.Info("HTTP 服务器已成功关闭")
	}

	// TracerProvider 的关闭已通过 defer 处理

	logger.Info("服务已完全关闭")
}
