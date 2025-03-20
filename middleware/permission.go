package middleware

import (
	"net/http"
	"user_hub/constants"
	"user_hub/models/enums"

	"github.com/gin-gonic/gin"
)

// PermissionMiddleware 定义权限中间件，用于检查用户角色是否在允许的角色列表中
// - 输入: allowedRoles 可变参数，表示允许访问的角色列表
// - 输出: gin.HandlerFunc 中间件函数
func PermissionMiddleware(allowedRoles ...enums.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从上下文中获取用户状态
		// - 从 gin.Context 中读取 StatusContextKey 的值
		// - 如果不存在，返回禁止访问错误
		status, exists := c.Get(constants.StatusContextKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "状态获取失败"})
			return
		}

		// 2. 检查用户是否被拉黑
		// - 如果状态为 Blacklisted，禁止访问
		// - 返回用户被拉黑的错误响应
		if status == enums.Blacklisted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "用户已经被拉黑"})
			return
		}

		// 3. 从上下文中获取用户角色
		// - 从 gin.Context 中读取 RoleContextKey 的值
		// - 如果不存在，返回权限不足错误
		roleList, exists := c.Get(constants.RoleContextKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}

		// 4. 转换角色类型
		// - 将 roleList 转换为 enums.UserRole 类型
		// - 如果转换失败，返回角色无效错误
		roleUUID, ok := roleList.(enums.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "角色信息无效"})
			return
		}

		// 5. 检查用户角色是否在允许列表中
		// - 遍历 allowedRoles，比较用户角色与允许的角色
		// - 如果匹配，继续处理请求
		for _, allowedRole := range allowedRoles {
			if roleUUID == allowedRole {
				c.Next()
				return
			}
		}

		// 6. 角色不匹配，拒绝访问
		// - 如果用户角色不在允许列表中，返回权限不足错误
		// - 使用 HTTP 状态码 403 表示禁止访问
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
	}
}
