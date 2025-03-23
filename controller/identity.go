package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv" // 新增，用于字符串转换
	"user_hub/common/code"
	"user_hub/common/dependencies"
	"user_hub/common/response"
	"user_hub/middleware"
	"user_hub/models/dto"
	"user_hub/models/enums"
	"user_hub/models/vo"
	service "user_hub/service/identity"
	"user_hub/userError"
)

// IdentityController 身份管理控制器
type IdentityController struct {
	identityService service.IdentityService          // 身份服务实例
	jwtUtil         dependencies.JWTUtilityInterface // JWT依赖
}

// NewIdentityController 创建 IdentityController 实例
// - 输入: identityService 身份服务实例
// - 输出: *IdentityController 控制器实例
func NewIdentityController(identityService service.IdentityService, jwtUtil dependencies.JWTUtilityInterface) *IdentityController {
	return &IdentityController{
		identityService: identityService,
		jwtUtil:         jwtUtil,
	}
}

// CreateIdentityHandler 处理创建新身份请求
// @Summary 创建新身份
// @Description 用户绑定新登录方式
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param body body dto.CreateIdentityDTO true "创建身份请求"
// @Success 200 {object} response.APIResponse[vo.IdentityVO] "身份创建成功"
// @Failure 400 {object} response.APIResponse[any] "请求数据无效"
// @Failure 500 {object} response.APIResponse[any] "创建身份失败"
// @Router /api/v1/identities [post]
func (ctrl *IdentityController) CreateIdentityHandler(c *gin.Context) {
	// 1. 绑定请求数据
	var createIdentityDTO dto.CreateIdentityDTO
	if err := c.ShouldBindJSON(&createIdentityDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 2. 调用服务层创建身份
	identityVO, err := ctrl.identityService.CreateIdentity(c.Request.Context(), &createIdentityDTO)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "创建身份失败")
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess(c, identityVO, "身份创建成功")
}

// UpdateIdentityHandler 处理更新身份请求
// @Summary 更新身份信息
// @Description 用户修改密码等操作
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param identityID path string true "身份 ID"
// @Param body body dto.UpdateIdentityDTO true "更新身份请求"
// @Success 200 {object} response.APIResponse[vo.IdentityVO] "身份更新成功"
// @Failure 400 {object} response.APIResponse[any] "请求数据无效"
// @Failure 404 {object} response.APIResponse[any] "该身份不存在"
// @Failure 500 {object} response.APIResponse[any] "更新身份失败"
// @Router /api/v1/identities/{identityID} [put]
func (ctrl *IdentityController) UpdateIdentityHandler(c *gin.Context) {
	// 1. 获取路径参数并转换为 uint
	identityIDStr := c.Param("identityID")
	if identityIDStr == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 不能为空")
		return
	}
	identityID, err := strconv.ParseUint(identityIDStr, 10, 64) // 转换为 uint64
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 格式无效")
		return
	}

	// 2. 绑定请求数据
	var updateIdentityDTO dto.UpdateIdentityDTO
	if err := c.ShouldBindJSON(&updateIdentityDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 3. 调用服务层更新身份，使用 uint 类型
	identityVO, err := ctrl.identityService.UpdateIdentity(c.Request.Context(), uint(identityID), &updateIdentityDTO)
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该身份不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "更新身份失败")
		}
		return
	}

	// 4. 返回成功响应
	response.RespondSuccess(c, identityVO, "身份更新成功")
}

// DeleteIdentityHandler 处理删除身份请求
// @Summary 删除身份
// @Description 用户注销某个身份
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param identityID path string true "身份 ID"
// @Success 200 {object} response.APIResponse[any] "身份删除成功"
// @Failure 400 {object} response.APIResponse[any] "身份 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "该身份不存在"
// @Failure 500 {object} response.APIResponse[any] "删除身份失败"
// @Router /api/v1/identities/{identityID} [delete]
func (ctrl *IdentityController) DeleteIdentityHandler(c *gin.Context) {
	// 1. 获取路径参数并转换为 uint
	identityIDStr := c.Param("identityID")
	if identityIDStr == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 不能为空")
		return
	}
	identityID, err := strconv.ParseUint(identityIDStr, 10, 64) // 转换为 uint64
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 格式无效")
		return
	}

	// 2. 调用服务层删除身份，使用 uint 类型
	err = ctrl.identityService.DeleteIdentity(c.Request.Context(), uint(identityID))
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该身份不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "删除身份失败")
		}
		return
	}

	// 3. 返回成功响应
	response.RespondSuccess[interface{}](c, nil, "身份删除成功")
}

// GetIdentitiesByUserIDHandler 处理获取用户身份请求
// @Summary 获取用户身份
// @Description 管理员查看用户的所有身份信息
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[vo.IdentityList] "获取身份信息成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "该用户不存在"
// @Failure 500 {object} response.APIResponse[any] "获取身份信息失败"
// @Router /api/v1/users/{userID}/identities [get]
func (ctrl *IdentityController) GetIdentitiesByUserIDHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取身份
	identitiesVO, err := ctrl.identityService.GetIdentitiesByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "获取身份信息失败")
		}
		return
	}

	responseData := vo.IdentityList{Items: identitiesVO}

	// 3. 返回成功响应
	response.RespondSuccess(c, responseData, "获取身份信息成功")
}

// GetIdentityTypesByUserIDHandler 处理获取用户身份类型请求
// @Summary 获取用户身份类型
// @Description 用户查看自己绑定的身份类型
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[vo.IdentityTypeList] "获取身份类型成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "该用户不存在"
// @Failure 500 {object} response.APIResponse[any] "获取身份类型失败"
// @Router /api/v1/users/{userID}/identity-types [get]
func (ctrl *IdentityController) GetIdentityTypesByUserIDHandler(c *gin.Context) {
	// 1. 获取路径参数
	userID := c.Param("userID")
	if userID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取身份类型
	identityTypes, err := ctrl.identityService.GetIdentityTypesByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			response.RespondError(c, http.StatusNotFound, code.ErrCodeClientResourceNotFound, "该用户不存在")
		} else {
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "获取身份类型失败")
		}
		return
	}

	responseData := vo.IdentityTypeList{Items: identityTypes}

	// 3. 返回成功响应
	response.RespondSuccess(c, responseData, "获取身份类型成功")
}

// RegisterRoutes 注册 IdentityController 的路由
// 该方法将所有与身份管理相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 根据需求为每个路由添加认证和权限中间件，确保访问控制符合规划
func (ctrl *IdentityController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 identities 子路由组
	// - 意图: 处理身份相关的操作（如创建、更新、删除），路径前缀为 /identities
	// - 路径: /api/v1/identities
	identities := group.Group("/identities")
	{
		// 第二步：注册创建身份路由
		// - 意图: 允许认证后的管理员绑定新身份，需要验证令牌并限制为 Admin 角色
		// - 方法: POST /api/v1/identities
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		identities.POST("", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.CreateIdentityHandler)

		// 第三步：注册更新身份路由
		// - 意图: 允许认证后的管理员或用户修改身份信息，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: PUT /api/v1/identities/{identityID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		identities.PUT("/:identityID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.UpdateIdentityHandler)

		// 第四步：注册删除身份路由
		// - 意图: 允许认证后的管理员或用户注销身份，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: DELETE /api/v1/identities/{identityID}
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		identities.DELETE("/:identityID", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.DeleteIdentityHandler)
	}

	// 第五步：创建 users 子路由组
	// - 意图: 处理用户身份信息的查询，路径前缀为 /users
	// - 路径: /api/v1/users
	users := group.Group("/users")
	{
		// 第六步：注册获取用户身份路由
		// - 意图: 允许认证后的管理员查看指定用户的所有身份信息，需要验证令牌并限制为 Admin 角色
		// - 方法: GET /api/v1/users/{userID}/identities
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 限制为 Admin
		users.GET("/:userID/identities", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin), ctrl.GetIdentitiesByUserIDHandler)

		// 第七步：注册获取用户身份类型路由
		// - 意图: 允许认证后的管理员或用户查看身份类型，需要验证令牌并限制为 Admin 或 User 角色
		// - 方法: GET /api/v1/users/{userID}/identity-types
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		users.GET("/:userID/identity-types", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.GetIdentityTypesByUserIDHandler)
	}
}
