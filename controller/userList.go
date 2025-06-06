package controller

import (
	"errors"
	"net/http"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/dependencies"
	// "user_hub/docs" // 如果您的 linter/IDE 需要，可以导入 docs 包，swag 通常会自动处理
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	service "github.com/Xushengqwer/user_hub/service/userList" // 假设 service/userList 包下有 UserListQueryService
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// UserListQueryController 处理用户列表查询相关的 HTTP 请求。
// 主要用于管理员后台等场景，提供带条件的用户列表分页查询功能。
type UserListQueryController struct {
	queryService service.UserListQueryService   // queryService: 用户列表查询服务的实例。
	jwtUtil      dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger       *core.ZapLogger                // logger: 日志记录器。
}

// NewUserListQueryController 创建一个新的 UserListQueryController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - queryService: 实现了 service.UserListQueryService 接口的服务实例。
//   - jwtUtil: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *UserListQueryController: 初始化完成的控制器实例。
func NewUserListQueryController(
	queryService service.UserListQueryService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) *UserListQueryController {
	return &UserListQueryController{
		queryService: queryService,
		jwtUtil:      jwtUtil,
		logger:       logger, // 存储 logger
	}
}

// ListUsersWithProfileHandler 处理分页查询用户及其关联 Profile 信息的请求。
// @Summary 分页查询用户及其资料 (管理员)
// @Description 管理员根据指定的过滤、排序和分页条件，查询用户列表及其关联的 Profile 信息。
// @Tags 用户查询 (User Query)
// @Accept json
// @Produce json
// @Param body body dto.UserQueryDTO true "查询条件 (过滤、排序、分页)"
// @Success 200 {object} docs.SwaggerAPIUserListResponse "查询成功，返回用户列表和总记录数"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、分页参数超出范围)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/user-hub/users/query [post] // <--- 已更新路径
func (ctrl *UserListQueryController) ListUsersWithProfileHandler(c *gin.Context) {
	const operation = "UserListQueryController.ListUsersWithProfileHandler"

	// 1. 绑定并校验请求体数据。
	var queryDTO dto.UserQueryDTO
	if err := c.ShouldBindJSON(&queryDTO); err != nil {
		ctrl.logger.Warn("查询用户列表请求参数绑定失败", zap.String("operation", operation), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}
	// 可以在此添加对 DTO 中 Filters, OrderBy 等字段更细致的校验逻辑（如果需要）
	// 但主要的安全性校验（如允许哪些字段过滤/排序）已在仓库层处理。

	// 2. 调用服务层执行查询逻辑。
	//    服务层会调用仓库层的 JoinQuery 来执行数据库查询。
	users, total, err := ctrl.queryService.ListUsersWithProfile(c.Request.Context(), &queryDTO)
	if err != nil {
		// 根据服务层返回的错误类型记录日志并响应。
		// UserListQueryService 通常只在数据库层面失败，返回 ErrSystemError。
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("查询用户列表服务返回系统错误",
				zap.String("operation", operation),
				zap.Any("queryDTO", queryDTO), // 记录查询参数
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他未预期的错误也视为系统错误。
			ctrl.logger.Error("查询用户列表服务返回未知错误",
				zap.String("operation", operation),
				zap.Any("queryDTO", queryDTO),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "查询用户列表时发生未知错误")
		}
		return
	}

	// 3. 构造响应数据。
	//    服务层直接返回了 vo.UserWithProfileVO 列表，无需控制器再次转换。
	responseData := vo.UserListResponse{
		Users: users,
		Total: total,
	}

	// 4. 记录日志并返回成功响应。
	ctrl.logger.Info("成功查询用户列表及其Profile信息",
		zap.String("operation", operation),
		zap.Int64("totalRecords", total),
		zap.Int("returnedRecords", len(users)),
		zap.Int("page", queryDTO.Page),
		zap.Int("pageSize", queryDTO.PageSize),
	)
	response.RespondSuccess(c, responseData, "查询成功")
}

// RegisterRoutes 注册与用户列表查询相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理此控制器的 API 端点。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例。如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/users" 子分组的完整基础路径将是 "/user-hub/api/v1/users"。
func (ctrl *UserListQueryController) RegisterRoutes(group *gin.RouterGroup) {
	// 在 /users 路径下注册查询接口
	// 使用 POST 方法接收复杂的查询条件 DTO
	usersRoutes := group.Group("/users")
	{
		// 注册分页查询用户及其资料的路由
		// - 场景: 管理员后台分页查看用户列表，支持筛选和排序。
		// - 预期权限: 需要认证，且角色为管理员 (Admin)，由网关处理。
		usersRoutes.POST("/query", ctrl.ListUsersWithProfileHandler)
	}
}
