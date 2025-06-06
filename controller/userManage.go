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
	service "github.com/Xushengqwer/user_hub/service/userManage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserManageController 处理与核心用户账户管理相关的 HTTP 请求。
// 例如：管理员创建用户、获取/更新/删除用户信息、拉黑用户等。
type UserManageController struct {
	userService service.UserManageService      // userService: 用户管理服务的实例。
	jwtToken    dependencies.JWTTokenInterface // jwtToken: JWT 工具，用于认证中间件。
	logger      *core.ZapLogger                // logger: 日志记录器。
}

// NewUserController 创建一个新的 UserManageController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - userService: 实现了 service.UserManageService 接口的服务实例。
//   - jwtUtil: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *UserManageController: 初始化完成的控制器实例。
func NewUserController( // 函数名保持与您提供的一致
	userService service.UserManageService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) *UserManageController {
	return &UserManageController{
		userService: userService,
		jwtToken:    jwtUtil,
		logger:      logger, // 存储 logger
	}
}

// CreateUserHandler 处理管理员创建新用户的请求。
// @Summary 创建新用户 (管理员)
// @Description 管理员根据提供的角色和状态信息创建一个新的用户账户。用户ID由系统自动生成。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param body body dto.CreateUserDTO true "创建用户请求，包含用户角色和初始状态"
// @Success 200 {object} docs.SwaggerAPIUserVOResponse "用户创建成功，返回新创建的用户信息"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、角色或状态值无效)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/user-hub/users [post] // <--- 已更新路径
func (ctrl *UserManageController) CreateUserHandler(c *gin.Context) {
	const operation = "UserManageController.CreateUserHandler"

	// 1. 绑定并校验请求体数据。
	var createUserDTO dto.CreateUserDTO
	if err := c.ShouldBindJSON(&createUserDTO); err != nil {
		ctrl.logger.Warn("创建新用户请求参数绑定失败", zap.String("operation", operation), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 2. 调用服务层执行创建用户的逻辑。
	userVO, err := ctrl.userService.CreateUser(c.Request.Context(), &createUserDTO)
	if err != nil {
		// CreateUser 服务通常只在数据库层面失败，返回 ErrSystemError
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他未预期的错误也视为系统错误
			ctrl.logger.Error("创建用户服务返回未知错误", zap.String("operation", operation), zap.Error(err))
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "创建用户时发生未知错误")
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功创建新用户",
		zap.String("operation", operation),
		zap.String("userID", userVO.UserID),
		zap.Any("role", userVO.UserRole),
		zap.Any("status", userVO.Status),
	)
	response.RespondSuccess(c, userVO, "用户创建成功")
}

// GetUserByIDHandler 处理根据用户ID获取核心用户信息的请求。
// @Summary 获取用户信息
// @Description 根据提供的用户ID获取该用户的核心账户信息（角色、状态、创建/更新时间等）。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param userID path string true "要查询的用户ID"
// @Success 200 {object} docs.SwaggerAPIUserVOResponse "获取用户信息成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员或用户本人)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/user-hub/users/{userID} [get] // <--- 已更新路径
func (ctrl *UserManageController) GetUserByIDHandler(c *gin.Context) {
	const operation = "UserManageController.GetUserByIDHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("获取用户信息请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}
	// 权限校验：是否是用户本人或管理员？依赖中间件。

	// 2. 调用服务层获取用户信息。
	userVO, err := ctrl.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "用户不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误或未预期的服务层错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功获取用户信息", zap.String("operation", operation), zap.String("userID", userID))
	response.RespondSuccess(c, userVO) // 成功时通常不需要 message
}

// GetUserProfileByAdminHandler 处理管理员根据用户ID获取指定用户详细资料的请求。
// @Summary 获取指定用户资料 (管理员)
// @Description (管理员权限) 根据提供的用户ID，获取该用户的详细个人资料信息（昵称、头像等）。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param userID path string true "要查询的用户ID"
// @Success 200 {object} docs.SwaggerAPIProfileVOResponse "获取用户资料成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定用户的资料不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/user-hub/users/{userID}/profile [get]
func (ctrl *UserManageController) GetUserProfileByAdminHandler(c *gin.Context) {
	const operation = "UserManageController.GetUserProfileByAdminHandler"
	targetUserID := c.Param("userID")
	if targetUserID == "" {
		ctrl.logger.Warn("管理员获取用户资料请求的目标用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "目标用户 ID 不能为空")
		return
	}

	profileVO, err := ctrl.userService.GetUserProfileByAdmin(c.Request.Context(), targetUserID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("管理员获取用户资料服务层返回系统错误", zap.String("operation", operation), zap.String("targetUserID", targetUserID), zap.Error(err))
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "指定用户的资料不存在" { // 匹配服务层返回的业务错误
			ctrl.logger.Info("管理员尝试获取不存在的用户资料", zap.String("operation", operation), zap.String("targetUserID", targetUserID))
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else { // 其他业务逻辑错误
			ctrl.logger.Warn("管理员获取用户资料服务层返回业务错误", zap.String("operation", operation), zap.String("targetUserID", targetUserID), zap.Error(err))
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	ctrl.logger.Info("管理员成功获取用户资料",
		zap.String("operation", operation),
		zap.String("targetUserID", targetUserID),
	)
	response.RespondSuccess(c, profileVO, "获取用户资料成功")
}

// UpdateUserHandler 处理更新用户核心信息（角色、状态）的请求。
// @Summary 更新用户信息 (管理员)
// @Description 管理员更新指定用户的角色和状态。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param userID path string true "要更新的用户ID"
// @Param body body dto.UpdateUserDTO true "包含待更新角色和/或状态的请求体"
// @Success 200 {object} docs.SwaggerAPIUserVOResponse "用户信息更新成功，返回更新后的用户信息"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、用户ID为空、角色或状态值无效)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/user-hub/users/{userID} [put] // <--- 已更新路径
func (ctrl *UserManageController) UpdateUserHandler(c *gin.Context) {
	const operation = "UserManageController.UpdateUserHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("更新用户信息请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 绑定并校验请求体数据。
	var updateUserDTO dto.UpdateUserDTO
	if err := c.ShouldBindJSON(&updateUserDTO); err != nil {
		ctrl.logger.Warn("更新用户信息请求参数绑定失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}
	// 可以在此添加对 DTO 中 Role 和 Status 枚举值的进一步校验（如果 binding 标签不够）

	// 3. 调用服务层执行更新逻辑。
	userVO, err := ctrl.userService.UpdateUser(c.Request.Context(), userID, &updateUserDTO)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要更新的用户不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 返回成功响应。
	ctrl.logger.Info("成功更新用户信息",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess(c, userVO, "用户信息更新成功")
}

// DeleteUserHandler 处理（软）删除用户的请求。
// @Summary 删除用户 (管理员)
// @Description 管理员（软）删除指定的用户账户及其所有关联数据（如身份、资料）。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param userID path string true "要删除的用户ID"
// @Success 200 {object} docs.SwaggerAPIEmptyResponse "用户删除成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户不存在 (如果服务层认为删除不存在的用户是错误)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库事务失败)"
// @Router /api/v1/user-hub/users/{userID} [delete] // <--- 已更新路径
func (ctrl *UserManageController) DeleteUserHandler(c *gin.Context) {
	const operation = "UserManageController.DeleteUserHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("删除用户请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层执行删除用户的逻辑（包含事务性删除关联数据）。
	err := ctrl.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要删除的用户不存在" { // 假设服务层对删除不存在用户返回此业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功删除用户及其关联数据",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess[vo.Empty](c, vo.Empty{}, "用户删除成功")
}

// BlackUserHandler 处理将用户加入黑名单的请求。
// @Summary 拉黑用户 (管理员)
// @Description 管理员将指定的用户账户状态设置为“拉黑”，阻止其登录或访问受限资源。
// @Tags 用户管理 (User Management)
// @Accept json
// @Produce json
// @Param userID path string true "要拉黑的用户ID"
// @Success 200 {object} docs.SwaggerAPIEmptyResponse "用户已成功拉黑"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 403 {object} docs.SwaggerAPIErrorResponseString "权限不足 (非管理员操作)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/user-hub/users/{userID}/blacklist [put] // <--- 已更新路径
func (ctrl *UserManageController) BlackUserHandler(c *gin.Context) {
	const operation = "UserManageController.BlackUserHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("拉黑用户请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层执行拉黑用户的逻辑。
	err := ctrl.userService.BlackUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要拉黑的用户不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功拉黑用户",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess[vo.Empty](c, vo.Empty{}, "用户已拉黑")
}

// RegisterRoutes 注册与核心用户管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理用户 CRUD 和状态变更的 API 端点。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例。如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/users" 子分组的完整基础路径将是 "/user-hub/api/v1/users"。
func (ctrl *UserManageController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /users 子路由组，用于核心用户管理操作
	usersRoutes := group.Group("/users")
	{
		// 创建新用户
		// - 场景: 管理员后台添加新用户账户。
		// - 预期权限: 需要认证，且角色为管理员 (Admin)。
		usersRoutes.POST("", ctrl.CreateUserHandler)

		// 获取用户信息
		// - 场景: 管理员查看用户详情，或用户查看自己的核心信息。
		// - 预期权限: 需要认证，管理员可查看所有用户，普通用户仅能查看自己 (需进行UserID匹配或角色判断)。
		usersRoutes.GET("/:userID", ctrl.GetUserByIDHandler)

		// 更新用户信息 (角色、状态)
		// - 场景: 管理员修改用户的角色或状态。
		// - 预期权限: 需要认证，且角色为管理员 (Admin)。
		usersRoutes.PUT("/:userID", ctrl.UpdateUserHandler)

		// 删除用户 (软删除)
		// - 场景: 管理员停用或删除用户账户。
		// - 预期权限: 需要认证，且角色为管理员 (Admin)。
		usersRoutes.DELETE("/:userID", ctrl.DeleteUserHandler)

		// 拉黑用户 (更新状态)
		// - 场景: 管理员将用户加入黑名单。
		// - 预期权限: 需要认证，且角色为管理员 (Admin)。
		usersRoutes.PUT("/:userID/blacklist", ctrl.BlackUserHandler)

		// 新增：管理员获取指定用户详细资料的路由
		usersRoutes.GET("/:userID/profile", ctrl.GetUserProfileByAdminHandler)
	}
}
