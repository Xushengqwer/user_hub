package router

import (
	"github.com/gin-gonic/gin"
	"user_hub/common/dependencies"
	"user_hub/controller"
	"user_hub/middleware"
	"user_hub/models/enums"
)

// SetupUserRoutes 设置用户相关路由
func SetupUserRoutes(router *gin.Engine, handlers *controller.UserAuthController, jwtUtil dependencies.JWTUtilityInterface) {
	// 定义用户模块的 API 前缀和版本
	apiV1 := router.Group("/api/v1")

	// 公共路由（无需认证）
	publicAuth := apiV1.Group("/auth")
	{
		// 密码登录
		publicAuth.POST("/login/password", handlers.LoginByPassword)
		// 退出登录
		publicAuth.DELETE("/logout", handlers.Logout)
		// 刷新令牌
		publicAuth.POST("/tokens/refresh", handlers.RefreshToken)
	}

	// 保护路由（需要认证）
	protected := apiV1.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtUtil))
	{
		// 获取当前用户信息
		protected.GET("/users/me", handlers.GetCurrentUser)

		// 用户管理路由（需要管理员权限）
		userGroup := protected.Group("/users")
		userGroup.Use(middleware.PermissionMiddleware(enums.Admin))
		{
			// 创建用户
			userGroup.POST("/create", handlers.CreateUser)
			// 查询用户列表
			userGroup.GET("/list", handlers.ListUsers)
			// 编辑用户
			userGroup.PUT("/update", handlers.UpdateUser)
			// 删除用户
			userGroup.DELETE("/:userID", handlers.DeleteUser)
			// 拉黑用户
			userGroup.PUT("/ban/:userID", handlers.BanUser)
		}
	}
}
