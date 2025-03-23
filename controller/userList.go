package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/dependencies"
	"user_hub/common/response"
	"user_hub/middleware"
	"user_hub/models/dto"
	"user_hub/models/enums"
	"user_hub/models/vo"
	service "user_hub/service/userList"
)

// QueryController 查询控制器
type QueryController struct {
	queryService service.QueryService             // 查询服务实例
	jwtUtil      dependencies.JWTUtilityInterface // JWT依赖
}

// NewQueryController 创建 QueryController 实例
// - 输入: queryService 查询服务实例
// - 输出: *QueryController 控制器实例
func NewQueryController(queryService service.QueryService, jwtUtil dependencies.JWTUtilityInterface) *QueryController {
	return &QueryController{
		queryService: queryService,
		jwtUtil:      jwtUtil,
	}
}

// ListUsersWithProfileHandler 处理分页查询用户及其资料请求
// @Summary 分页查询用户及其资料
// @Description 管理员查看用户列表及其资料，支持过滤、排序和分页
// @Tags 查询管理
// @Accept json
// @Produce json
// @Param body body dto.UserQueryDTO true "查询条件"
// // @Success 200 {object} response.APIResponse[vo.UserListResponse] "查询成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "查询失败"
// @Router /api/v1/users/query [post]
func (ctrl *QueryController) ListUsersWithProfileHandler(c *gin.Context) {
	// 1. 绑定请求数据
	var dto dto.UserQueryDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 调用服务层执行查询
	users, total, err := ctrl.queryService.ListUsersWithProfile(c.Request.Context(), &dto)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "查询失败")
		return
	}

	// 3. 构造响应数据
	// - 将用户列表和总数封装到结构体中
	responseData := vo.UserListResponse{
		Users: users,
		Total: total,
	}

	// 4. 返回成功响应
	response.RespondSuccess(c, responseData, "查询成功")
}

// RegisterRoutes 注册 QueryController 的路由
// 该方法将所有与查询管理相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 为查询路由添加认证和权限中间件，确保只有认证后的管理员可以访问
func (ctrl *QueryController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 users 子路由组
	// - 意图: 处理用户查询相关的操作，路径前缀为 /users，与身份查询路由保持一致
	// - 路径: /api/v1/users
	users := group.Group("/users")
	{
		// 第二步：注册分页查询用户及其资料路由
		// - 意图: 允许认证后的管理员查询用户列表及其资料，需要验证令牌并限制为 Admin 角色
		// - 方法: POST /api/v1/users/query
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.POST("/query", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.ListUsersWithProfileHandler)
	}
}
