package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/dependencies"
	"user_hub/common/response"
	"user_hub/middleware"
	"user_hub/models/dto"
	"user_hub/models/enums"
	service "user_hub/service/user"
	"user_hub/userError"
)

// UserController 用户管理控制器
type UserController struct {
	userService service.UserService              // 用户服务实例
	jwtUtil     dependencies.JWTUtilityInterface // JWT依赖
}

// NewUserController 创建 UserController 实例
// - 输入: userService 用户服务实例
// - 输出: *UserController 控制器实例
func NewUserController(userService service.UserService, jwtUtil dependencies.JWTUtilityInterface) *UserController {
	return &UserController{
		userService: userService,
		jwtUtil:     jwtUtil,
	}
}

// CreateUserHandler 处理创建新用户请求
// @Summary 创建新用户
// @Description 管理员创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param body body dto.CreateUserDTO true "创建用户请求"
// @Success 200 {object} response.APIResponse[vo.UserVO]
// @Failure 400 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /api/v1/users [post]
func (ctrl *UserController) CreateUserHandler(c *gin.Context) {
	// 1. 绑定请求数据
	var createUserDTO dto.CreateUserDTO
	if err := c.ShouldBindJSON(&createUserDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 2. 调用服务层创建用户
	userVO, err := ctrl.userService.CreateUser(c.Request.Context(), &createUserDTO)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, code.ErrCodeClientInvalidInput, "创建用户失败")
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess(c, userVO, "用户创建成功")
}

// GetUserByIDHandler 处理获取用户信息请求
// @Summary 获取用户信息
// @Description 根据用户 ID 获取用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[vo.UserVO]
// @Failure 404 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /api/v1/users/{userID} [get]
func (ctrl *UserController) GetUserByIDHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取用户
	userVO, err := ctrl.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeClientInvalidInput, "获取用户信息失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess(c, userVO)
}

// UpdateUserHandler 处理更新用户信息请求
// @Summary 更新用户信息
// @Description 管理员更新用户角色和状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Param body body dto.UpdateUserDTO true "更新用户请求"
// @Success 200 {object} response.APIResponse[vo.UserVO]
// @Failure 400 {object} response.APIResponse[any]
// @Failure 404 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /api/v1/users/{userID} [put]
func (ctrl *UserController) UpdateUserHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 绑定请求数据
	var updateUserDTO dto.UpdateUserDTO
	if err := c.ShouldBindJSON(&updateUserDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 3. 调用服务层更新用户
	userVO, err := ctrl.userService.UpdateUser(c.Request.Context(), userID, &updateUserDTO)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "更新用户失败")
		}
		return
	}

	// 4. 返回成功响应
	response.RespondSuccess(c, userVO, "用户更新成功")
}

// DeleteUserHandler 处理删除用户请求
// @Summary 删除用户
// @Description 管理员删除用户（软删除）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[any]
// @Failure 404 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /api/v1/users/{userID} [delete]
func (ctrl *UserController) DeleteUserHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层删除用户
	err := ctrl.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "删除用户失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess[interface{}](c, nil, "用户删除成功")
}

// BlackUserHandler 处理拉黑用户请求
// @Summary 拉黑用户
// @Description 管理员将用户加入黑名单
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[any]
// @Failure 404 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /api/v1/users/{userID}/blacklist [put]
func (ctrl *UserController) BlackUserHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层拉黑用户
	err := ctrl.userService.BlackUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "拉黑用户失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess[interface{}](c, nil, "用户已拉黑")
}

// RegisterRoutes 注册 UserController 的路由
// 该方法将所有与用户管理相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 为用户管理路由添加认证和权限中间件，确保访问控制符合需求
func (ctrl *UserController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 users 子路由组
	// - 意图: 处理用户相关的操作（如创建、获取、更新、删除、拉黑），路径前缀为 /users，与其他用户相关路由保持一致
	// - 路径: /api/v1/users
	users := group.Group("/users")
	{
		// 第二步：注册创建用户路由
		// - 意图: 允许认证后的管理员创建新用户，需要验证令牌并限制为 Admin 角色
		// - 方法: POST /api/v1/users
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.POST("", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.CreateUserHandler)

		// 第三步：注册获取用户信息路由
		// - 意图: 允许认证后的管理员或用户根据 ID 获取用户信息，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: GET /api/v1/users/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		users.GET("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.GetUserByIDHandler)

		// 第四步：注册更新用户信息路由
		// - 意图: 允许认证后的管理员更新用户信息，需要验证令牌并限制为 Admin 角色
		// - 方法: PUT /api/v1/users/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.PUT("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.UpdateUserHandler)

		// 第五步：注册删除用户路由
		// - 意图: 允许认证后的管理员删除用户，需要验证令牌并限制为 Admin 角色
		// - 方法: DELETE /api/v1/users/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.DELETE("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.DeleteUserHandler)

		// 第六步：注册拉黑用户路由
		// - 意图: 允许认证后的管理员将用户加入黑名单，需要验证令牌并限制为 Admin 角色
		// - 方法: PUT /api/v1/users/{userID}/blacklist
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.PUT("/:userID/blacklist", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.BlackUserHandler)
	}
}
