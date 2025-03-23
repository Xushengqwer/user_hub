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
	service "user_hub/service/profile"
	"user_hub/userError"
)

// ProfileController 资料管理控制器
type ProfileController struct {
	profileService service.ProfileService           // 资料服务实例
	jwtUtil        dependencies.JWTUtilityInterface // JWT依赖
}

// NewProfileController 创建 ProfileController 实例
// - 输入: profileService 资料服务实例
// - 输出: *ProfileController 控制器实例
func NewProfileController(profileService service.ProfileService, jwtUtil dependencies.JWTUtilityInterface) *ProfileController {
	return &ProfileController{
		profileService: profileService,
		jwtUtil:        jwtUtil,
	}
}

// CreateProfileHandler 处理创建用户资料请求
// @Summary 创建用户资料
// @Description 用户首次填写个人资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param body body dto.CreateProfileDTO true "创建资料请求"
// @Success 200 {object} response.APIResponse[vo.ProfileVO] "资料创建成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "创建资料失败"
// @Router /api/v1/profiles [post]
func (ctrl *ProfileController) CreateProfileHandler(c *gin.Context) {
	// 1. 绑定请求数据
	var createProfileDTO dto.CreateProfileDTO
	if err := c.ShouldBindJSON(&createProfileDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 调用服务层创建资料
	profileVO, err := ctrl.profileService.CreateProfile(c.Request.Context(), &createProfileDTO)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "创建资料失败")
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess(c, profileVO, "资料创建成功")
}

// GetProfileByUserIDHandler 处理获取用户资料请求
// @Summary 获取用户资料
// @Description 根据用户 ID 获取用户资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[vo.ProfileVO] "获取资料成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "用户资料为空"
// @Failure 500 {object} response.APIResponse[any] "获取资料失败"
// @Router /api/v1/profiles/{userID} [get]
func (ctrl *ProfileController) GetProfileByUserIDHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取资料
	profileVO, err := ctrl.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrProfileNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "用户资料为空")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "获取资料失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess(c, profileVO, "获取资料成功")
}

// UpdateProfileHandler 处理更新用户资料请求
// @Summary 更新用户资料
// @Description 用户或管理员更新用户资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Param body body dto.UpdateProfileDTO true "更新资料请求"
// @Success 200 {object} response.APIResponse[vo.ProfileVO] "资料更新成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 404 {object} response.APIResponse[any] "用户资料不存在"
// @Failure 500 {object} response.APIResponse[any] "更新资料失败"
// @Router /api/v1/profiles/{userID} [put]
func (ctrl *ProfileController) UpdateProfileHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 绑定请求数据
	var dto dto.UpdateProfileDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 3. 调用服务层更新资料
	profileVO, err := ctrl.profileService.UpdateProfile(c.Request.Context(), userID, &dto)
	if err != nil {
		if errors.Is(err, userError.ErrProfileNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "用户资料不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "更新资料失败")
		}
		return
	}

	// 4. 返回成功响应
	response.RespondSuccess(c, profileVO, "资料更新成功")
}

// DeleteProfileHandler 处理删除用户资料请求
// @Summary 删除用户资料
// @Description 用户或管理员删除用户资料
// @Tags 资料管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[any] "资料删除成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "用户资料不存在"
// @Failure 500 {object} response.APIResponse[any] "删除资料失败"
// @Router /api/v1/profiles/{userID} [delete]
func (ctrl *ProfileController) DeleteProfileHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层删除资料
	err := ctrl.profileService.DeleteProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrProfileNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "用户资料不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "删除资料失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess[interface{}](c, nil, "资料删除成功")
}

// RegisterRoutes 注册 ProfileController 的路由
// 该方法将所有与用户资料管理相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 为每个资料管理路由添加认证和权限中间件，确保只有认证后的管理员或用户可以访问
func (ctrl *ProfileController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 profiles 子路由组
	// - 意图: 处理用户资料相关的操作（如创建、获取、更新、删除），路径前缀为 /profiles
	// - 路径: /api/v1/profiles
	profiles := group.Group("/profiles")
	{
		// 第二步：注册创建用户资料路由
		// - 意图: 允许认证后的管理员或用户创建资料，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: POST /api/v1/profiles
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		profiles.POST("", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.CreateProfileHandler)

		// 第三步：注册获取用户资料路由
		// - 意图: 允许认证后的管理员或用户根据用户 ID 获取资料，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: GET /api/v1/profiles/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		profiles.GET("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.GetProfileByUserIDHandler)

		// 第四步：注册更新用户资料路由
		// - 意图: 允许认证后的管理员或用户更新资料，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: PUT /api/v1/profiles/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		profiles.PUT("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.UpdateProfileHandler)

		// 第五步：注册删除用户资料路由
		// - 意图: 允许认证后的管理员或用户删除资料，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: DELETE /api/v1/profiles/{userID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		profiles.DELETE("/:userID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.DeleteProfileHandler)
	}
}
