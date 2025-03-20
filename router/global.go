package router

//// 注册中间件  -- 顺序
//router.Use(middleware.ErrorHandlingMiddleware(logger))
//router.Use(middleware.RequestIDMiddleware())
//router.Use(middleware.RequestLoggerMiddleware(logger))
//router.Use(middleware.CorsMiddleware())
//router.Use(middleware.RateLimitMiddleware(logger, cfg))
//router.Use(middleware.RequestTimeoutMiddleware(logger, timeout))
//router.Use(middleware.AuthMiddleware(jwtUtil))
//router.Use(middleware.PermissionMiddleware(allowedRoles...))

//这个顺序确保了：
//
//全局功能（如错误处理、日志）覆盖所有请求。
//性能优化（如限流）尽早执行。
//依赖关系（如认证 -> 授权）正确满足。
