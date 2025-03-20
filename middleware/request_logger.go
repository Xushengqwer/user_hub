package middleware

import (
	"time"
	"user_hub/common/core"
	"user_hub/constants"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLoggerMiddleware 定义请求日志中间件，用于记录每个请求的关键信息
// - 输入: logger ZapLogger 实例，用于记录日志
// - 输出: gin.HandlerFunc 中间件函数
func RequestLoggerMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 记录请求开始时间
		// - 使用 time.Now() 获取当前时间，作为请求处理的起点
		startTime := time.Now()

		// 2. 处理后续请求
		// - 调用 c.Next() 执行后续中间件或控制器逻辑
		c.Next()

		// 3. 计算处理时长
		// - 获取请求结束时间并计算与开始时间的差值
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 4. 从上下文中获取请求信息
		// - 提取请求方法、路径、状态码、客户端 IP 和用户代理
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		rid, _ := c.Get(constants.RequestIDKey)
		requestID, _ := rid.(string)

		// 5. 记录请求日志
		// - 使用 ZapLogger 记录请求的详细信息
		// - 包括请求 ID、方法、路径、状态码、客户端 IP、用户代理和处理时长
		logger.Info("HTTP 请求",
			zap.String("request_id", requestID), // 请求唯一标识
			zap.String("method", method),        // 请求方法（如 GET、POST）
			zap.String("path", path),            // 请求路径
			zap.Int("status", statusCode),       // 响应状态码
			zap.String("client_ip", clientIP),   // 客户端 IP 地址
			zap.String("user_agent", userAgent), // 用户代理信息
			zap.Duration("latency", latency),    // 请求处理时长
		)
	}
}
