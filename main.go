package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv" // 新增：用于将字符串转换为布尔值或整数
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

	// --- 手动从环境变量覆盖关键配置 (生产环境部署核心) ---
	log.Println("检查环境变量以覆盖 User Hub 的文件配置...")
	// 遍历并覆盖所有通过环境变量注入的配置
	// Server & Log
	if level := os.Getenv("ZAPCONFIG_LEVEL"); level != "" {
		cfg.ZapConfig.Level = level
		log.Printf("通过环境变量覆盖了 ZapConfig.Level: %s\n", level)
	}
	if level := os.Getenv("GORMLOGCONFIG_LEVEL"); level != "" {
		cfg.GormLogConfig.Level = level
		log.Printf("通过环境变量覆盖了 GormLogConfig.Level: %s\n", level)
	}
	// Tracer
	if enabled, err := strconv.ParseBool(os.Getenv("TRACERCONFIG_ENABLED")); err == nil {
		cfg.TracerConfig.Enabled = enabled
		log.Printf("通过环境变量覆盖了 TracerConfig.Enabled: %t\n", enabled)
	}
	// JWT
	if key := os.Getenv("JWTCONFIG_SECRET_KEY"); key != "" {
		cfg.JWTConfig.SecretKey = key
		log.Printf("通过环境变量覆盖了 JWTConfig.SecretKey") // 不打印密钥值
	}
	if key := os.Getenv("JWTCONFIG_REFRESH_SECRET"); key != "" {
		cfg.JWTConfig.RefreshSecret = key
		log.Printf("通过环境变量覆盖了 JWTConfig.RefreshSecret") // 不打印密钥值
	}
	// MySQL & Redis
	if dsn := os.Getenv("MYSQLCONFIG_DSN"); dsn != "" {
		cfg.MySQLConfig.DSN = dsn
		log.Printf("通过环境变量覆盖了 MySQLConfig.DSN") // 不打印DSN
	}
	if addr := os.Getenv("REDISCONFIG_ADDRESS"); addr != "" {
		cfg.RedisConfig.Address = addr
		log.Printf("通过环境变量覆盖了 RedisConfig.Address: %s\n", addr)
	}
	if pass := os.Getenv("REDISCONFIG_PASSWORD"); pass != "" {
		cfg.RedisConfig.Password = pass
		log.Printf("通过环境变量覆盖了 RedisConfig.Password")
	}

	// COS
	if id := os.Getenv("COSCONFIG_SECRET_ID"); id != "" {
		cfg.COSConfig.SecretID = id
		log.Printf("通过环境变量覆盖了 CosConfig.SecretId")
	}
	if key := os.Getenv("COSCONFIG_SECRET_KEY"); key != "" {
		cfg.COSConfig.SecretKey = key
		log.Printf("通过环境变量覆盖了 CosConfig.SecretKey")
	}
	if name := os.Getenv("COSCONFIG_BUCKET_NAME"); name != "" {
		cfg.COSConfig.BucketName = name
		log.Printf("通过环境变量覆盖了 CosConfig.BucketName: %s\n", name)
	}
	if id := os.Getenv("COSCONFIG_APP_ID"); id != "" {
		cfg.COSConfig.AppID = id
		log.Printf("通过环境变量覆盖了 CosConfig.AppId: %s\n", id)
	}
	if region := os.Getenv("COSCONFIG_REGION"); region != "" {
		cfg.COSConfig.Region = region
		log.Printf("通过环境变量覆盖了 CosConfig.Region: %s\n", region)
	}
	if url := os.Getenv("COSCONFIG_BASE_URL"); url != "" {
		cfg.COSConfig.BaseURL = url
		log.Printf("通过环境变量覆盖了 CosConfig.BaseURL: %s\n", url)
	}
	// Cookie
	if secure, err := strconv.ParseBool(os.Getenv("COOKIECONFIG_SECURE")); err == nil {
		cfg.CookieConfig.Secure = secure
		log.Printf("通过环境变量覆盖了 CookieConfig.Secure: %t\n", secure)
	}
	if domain := os.Getenv("COOKIECONFIG_DOMAIN"); domain != "" {
		cfg.CookieConfig.Domain = domain
		log.Printf("通过环境变量覆盖了 CookieConfig.Domain: %s\n", domain)
	}
	if name := os.Getenv("COOKIECONFIG_REFRESH_TOKEN_NAME"); name != "" {
		cfg.CookieConfig.RefreshTokenName = name
		log.Printf("通过环境变量覆盖了 CookieConfig.RefreshTokenName: %s\n", name)
	}
	// --- 结束环境变量覆盖 ---

	// 2. 初始化 Logger (使用可能已被覆盖的配置)
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

	// ... (main 函数的其余部分保持不变) ...
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

	// 8. 启动服务器
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
