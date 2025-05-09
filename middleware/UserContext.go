package middleware

import (
	"github.com/gin-gonic/gin"
)

func UserContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 HTTP 头获取用户信息
		userID := c.GetHeader("X-User-ID")
		role := c.GetHeader("X-User-Role")
		status := c.GetHeader("X-User-Status")
		platform := c.GetHeader("X-Platform")

		// 存入 Context
		c.Set("UserID", userID)
		c.Set("Role", role)
		c.Set("Status", status)
		c.Set("Platform", platform)

		// 继续处理请求
		c.Next()
	}
}
