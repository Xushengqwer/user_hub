package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	service "user_hub/service/profile"
	"user_hub/userError"
)

// ProfileController 资料管理控制器
type ProfileController struct {
	profileService service.ProfileService // 资料服务实例
}

// NewProfileController 创建 ProfileController 实例
// - 输入: profileService 资料服务实例
// - 输出: *ProfileController 控制器实例
func NewProfileController(profileService service.ProfileService) *ProfileController {
	return &ProfileController{profileService: profileService}
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
// @Router /profiles [post]
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
// @Router /profiles/{userID} [get]
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
// @Router /profiles/{userID} [put]
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
// @Router /profiles/{userID} [delete]
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
