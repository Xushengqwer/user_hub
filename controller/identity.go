package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	service "user_hub/service/identity"
	"user_hub/userError"
)

// IdentityController 身份管理控制器
type IdentityController struct {
	identityService service.IdentityService // 身份服务实例
}

// NewIdentityController 创建 IdentityController 实例
// - 输入: identityService 身份服务实例
// - 输出: *IdentityController 控制器实例
func NewIdentityController(identityService service.IdentityService) *IdentityController {
	return &IdentityController{identityService: identityService}
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
// @Router /identities [post]
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
// @Router /identities/{identityID} [put]
func (ctrl *IdentityController) UpdateIdentityHandler(c *gin.Context) {
	// 1. 获取路径参数
	identityID := c.Param("identityID")
	if identityID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 不能为空")
		return
	}

	// 2. 绑定请求数据
	var updateIdentityDTO dto.UpdateIdentityDTO
	if err := c.ShouldBindJSON(&updateIdentityDTO); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 3. 调用服务层更新身份
	identityVO, err := ctrl.identityService.UpdateIdentity(c.Request.Context(), identityID, &updateIdentityDTO)
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
// @Router /identities/{identityID} [delete]
func (ctrl *IdentityController) DeleteIdentityHandler(c *gin.Context) {
	// 1. 获取路径参数
	identityID := c.Param("identityID")
	if identityID == "" {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "身份 ID 不能为空")
		return
	}

	// 2. 调用服务层删除身份
	err := ctrl.identityService.DeleteIdentity(c.Request.Context(), identityID)
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
// @Success 200 {object} response.APIResponse[[]vo.IdentityVO] "获取身份信息成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "该用户不存在"
// @Failure 500 {object} response.APIResponse[any] "获取身份信息失败"
// @Router /users/{userID}/identities [get]
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

	// 3. 返回成功响应
	response.RespondSuccess(c, identitiesVO, "获取身份信息成功")
}

// GetIdentityTypesByUserIDHandler 处理获取用户身份类型请求
// @Summary 获取用户身份类型
// @Description 用户查看自己绑定的身份类型
// @Tags 身份管理
// @Accept json
// @Produce json
// @Param userID path string true "用户 ID"
// @Success 200 {object} response.APIResponse[[]enums.IdentityType] "获取身份类型成功"
// @Failure 400 {object} response.APIResponse[any] "用户 ID 不能为空"
// @Failure 404 {object} response.APIResponse[any] "该用户不存在"
// @Failure 500 {object} response.APIResponse[any] "获取身份类型失败"
// @Router /users/{userID}/identity-types [get]
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

	// 3. 返回成功响应
	response.RespondSuccess(c, identityTypes, "获取身份类型成功")
}
