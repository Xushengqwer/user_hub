package main

import (
	"context"
	"encoding/json"
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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	// 导入项目包
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	_ "github.com/Xushengqwer/user_hub/docs"
	"github.com/Xushengqwer/user_hub/initialization"
	"github.com/Xushengqwer/user_hub/router"
)

// @title           User Hub API
// @version         1.0
// @description     用户中心服务 API 文档
// @termsOfService  http://swagger.io/terms/

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8081
// @schemes http https
func main() {
	// --- 配置和基础设置 ---
	var configFile string
	flag.StringVar(&configFile, "config", "config/config.development.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.UserHubConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("FATAL: 加载配置失败 (%s): %v", configFile, err)
	}

	// --- [新增] 打印最终生效的配置以供调试 ---
	configBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("无法序列化配置以进行打印: %v", err)
	}
	log.Printf("✅ 配置加载成功！最终生效的配置如下:\n%s\n", string(configBytes))

	// 2. 初始化 Logger
	logger, loggerErr := sharedCore.NewZapLogger(cfg.ZapConfig)
	if loggerErr != nil {
		log.Fatalf("FATAL: 初始化 ZapLogger 失败: %v", loggerErr)
	}
	defer func() {
		logger.Info("正在同步日志...")
		if err := logger.Logger().Sync(); err != nil {
			log.Printf("WARN: ZapLogger Sync 失败: %v\n", err)
		}
	}()
	logger.Info("Logger 初始化成功")

	// ... (main 函数的其余部分保持不变，从 Tracer 初始化到服务关闭) ...
	// 3. 初始化 TracerProvider (如果启用)
	var tracerShutdown func(context.Context) error = func(ctx context.Context) error { return nil }
	if cfg.TracerConfig.Enabled {
		var err error
		tracerShutdown, err = sharedTracing.InitTracerProvider(
			constants.ServiceName,
			constants.ServiceVersion,
			cfg.TracerConfig,
		)
		if err != nil {
			logger.Fatal("初始化 TracerProvider 失败", zap.Error(err))
		}
		defer func() {
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
		_ = otelhttp.NewTransport(http.DefaultTransport)
		logger.Debug("OTel HTTP Transport 初始化完成 (暂未使用)")
	} else {
		logger.Info("分布式追踪已禁用")
	}

	// 4. 初始化基础依赖 (数据库, Redis, JWT, 外部客户端等)
	appDeps, err := initialization.SetupDependencies(&cfg, logger)
	if err != nil {
		logger.Fatal("初始化基础依赖失败", zap.Error(err))
	}
	logger.Info("基础依赖初始化成功")

	// 5. 初始化服务层实例
	appServices := initialization.SetupServices(appDeps)
	logger.Info("服务层初始化成功")

	// 6. 设置路由和中间件
	setupRouter := router.SetupRouter(
		logger,
		&cfg,
		appDeps.JwtToken,
		appServices,
		appDeps,
	)
	logger.Info("Gin 路由器设置完成")

	// 7. 配置并启动 HTTP 服务器
	serverAddress := fmt.Sprintf(":%s", cfg.ServerConfig.Port)
	srv := &http.Server{
		Addr:    serverAddress,
		Handler: otelhttp.NewHandler(setupRouter, "HTTPServer"),
	}

	// 8. 启动服务器 (使用 goroutine，以便不阻塞后续的优雅关停逻辑)
	go func() {
		logger.Info("HTTP 服务器开始监听", zap.String("address", serverAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// 9. 等待中断信号以实现优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	recSignal := <-quit
	logger.Info("接收到关停信号", zap.String("signal", recSignal.String()))

	// 10. 执行优雅关停
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	logger.Info("开始优雅关停 HTTP 服务器...")
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("HTTP 服务器优雅关停失败", zap.Error(err))
	} else {
		logger.Info("HTTP 服务器已成功关闭")
	}

	logger.Info("服务已完全关闭")
}
