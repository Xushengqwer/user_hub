package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"user_hub/constants"
)

// RequestIDMiddleware 定义请求 ID 中间件，为每个请求生成或获取唯一 ID
// - 输出: gin.HandlerFunc 中间件函数
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查请求头中的 X-Request-Id
		// - 获取客户端或网关可能已生成的请求 ID
		// - 如果存在，则复用该 ID
		requestID := c.Request.Header.Get("X-Request-Id")

		// 2. 生成新的请求 ID（如果需要）
		// - 如果请求头中没有 X-Request-Id，则生成一个新的 UUID
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 3. 存储请求 ID 到上下文
		// - 将 requestID 存入 gin.Context，使用 constants.RequestIDKey 作为键
		// - 便于后续中间件或控制器使用
		c.Set(constants.RequestIDKey, requestID)

		// 4. 设置响应头
		// - 将 requestID 写入响应头的 X-Request-Id 字段
		// - 方便客户端或日志系统追踪请求
		c.Header("X-Request-Id", requestID)

		// 5. 继续处理请求
		// - 调用 c.Next() 执行后续中间件或处理逻辑
		c.Next()
	}
}
