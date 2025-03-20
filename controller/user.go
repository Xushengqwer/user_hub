package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	service "user_hub/service/manage"
	"user_hub/userError"
)

// UserController 用户管理控制器
type UserController struct {
	userService service.UserService // 用户服务实例
}

// NewUserController 创建 UserController 实例
// - 输入: userService 用户服务实例
// - 输出: *UserController 控制器实例
func NewUserController(userService service.UserService) *UserController {
	return &UserController{userService: userService}
}

// CreateUserHandler 处理创建新用户请求
// @Summary 创建新用户
// @Description 管理员创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param body dto.CreateUserDTO true "创建用户请求"
// @Success 200 {object} response.APIResponse[vo.UserVO]
// @Failure 400 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /users [post]
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
// @Router /users/{userID} [get]
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
// @Param body dto.UpdateUserDTO true "更新用户请求"
// @Success 200 {object} response.APIResponse[vo.UserVO]
// @Failure 400 {object} response.APIResponse[any]
// @Failure 404 {object} response.APIResponse[any]
// @Failure 500 {object} response.APIResponse[any]
// @Router /users/{userID} [put]
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
// @Router /users/{userID} [delete]
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
// @Router /users/{userID}/blacklist [put]
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
