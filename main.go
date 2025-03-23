package main

import (
	"fmt"
	swaggerFiles "github.com/swaggo/files"     // swagger-files 包，提供 Swagger UI 的静态文件
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger 包，用于集成 Swagger UI
	"go.uber.org/zap"
	"user_hub/common/core"
	_ "user_hub/docs" // Swagger 文档
	"user_hub/initialization"
	"user_hub/router"
)

// @title Your API title
// @version 1.0
// @description Your API description
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// 1. 初始化配置
	doerConfig, err := core.ViperLoadConfig()
	if err != nil {
		fmt.Printf("配置加载失败: %v\n", err) // 打印详细的错误信息
		return
	}

	// 2. 初始化日志实例
	zapLogger, err := core.NewZapLogger(doerConfig.ZapConfig)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		return
	}
	defer func() {
		if err := zapLogger.Logger().Sync(); err != nil {
			zapLogger.Error("ZapLogger Sync 失败: %v", zap.Error(err))
		}
	}()

	// 3. 初始化基础依赖
	dependencies, err := initialization.SetupDependencies(doerConfig, zapLogger)
	if err != nil {
		zapLogger.Fatal("初始化基础依赖失败", zap.Error(err))
		return
	}

	// 4. 初始化服务层
	appServices := initialization.SetupServices(dependencies)

	// 5. 设置路由
	setupRouter := router.SetupRouter(
		zapLogger,
		&doerConfig.RateLimitConfig,
		dependencies.SMS,
		dependencies.JwtToken,
		appServices,
	)

	// 7.配置 Swagger UI
	setupRouter.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 6. 启动服务器
	if err := setupRouter.Run(":8080"); err != nil {
		zapLogger.Fatal("Failed to run server", zap.Error(err))
	}
}
