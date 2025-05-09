package controller

import (
	"errors"
	"net/http"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段

	"user_hub/dependencies"
	"user_hub/models/dto"
	"user_hub/models/vo"
	service "user_hub/service/profile"
)

// UserProfileController 处理与用户详细资料（Profile）相关的 HTTP 请求。
// 依赖于 service.UserProfileService 来执行核心业务逻辑。
type UserProfileController struct {
	profileService service.UserProfileService     // profileService: 用户资料管理服务的实例。
	jwtUtil        dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger         *core.ZapLogger                // logger: 日志记录器。
}

// NewUserProfileController 创建一个新的 UserProfileController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - profileService: 实现了 service.UserProfileService 接口的服务实例。
//   - jwtToken: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *UserProfileController: 初始化完成的控制器实例。
func NewUserProfileController(
	profileService service.UserProfileService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) *UserProfileController {
	return &UserProfileController{
		profileService: profileService,
		jwtUtil:        jwtUtil,
		logger:         logger, // 存储 logger
	}
}

// CreateProfileHandler 处理为用户首次创建个人资料的请求。
// @Summary 创建用户资料
// @Description 用户首次填写或系统自动为其创建个人基础资料（昵称、头像、地区等）。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Param body body dto.CreateProfileDTO true "创建资料请求，必须包含 userID"
// @Success 200 {object} docs.SwaggerAPIProfileVOResponse "资料创建成功，返回创建后的资料信息"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、userID缺失) 或 业务逻辑错误 (如该用户资料已存在)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/profiles [post]
func (ctrl *UserProfileController) CreateProfileHandler(c *gin.Context) {
	const operation = "UserProfileController.CreateProfileHandler"

	// 1. 绑定并校验请求体数据。
	var createProfileDTO dto.CreateProfileDTO
	if err := c.ShouldBindJSON(&createProfileDTO); err != nil {
		ctrl.logger.Warn("创建用户资料请求参数绑定失败", zap.String("operation", operation), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}
	// 可以在此添加对 UserID 的额外校验逻辑，例如从 JWT 或其他上下文中获取 UserID 并进行比对，
	// 以确保用户只能创建自己的资料（除非是管理员操作）。当前依赖于权限中间件。

	// 2. 调用服务层执行创建资料的逻辑。
	profileVO, err := ctrl.profileService.CreateProfile(c.Request.Context(), &createProfileDTO)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他视为业务逻辑错误（例如，"该用户的资料已存在，请使用更新操作"）
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功创建用户资料",
		zap.String("operation", operation),
		zap.String("userID", profileVO.UserID),
		// 可以考虑记录 profileVO.ID (如果 Profile 有主键)
	)
	response.RespondSuccess(c, profileVO, "资料创建成功")
}

// GetProfileByUserIDHandler 处理根据用户ID获取其详细资料的请求。
// @Summary 获取用户资料
// @Description 根据提供的用户ID，获取该用户的详细个人资料信息。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Param userID path string true "要查询的用户ID"
// @Success 200 {object} docs.SwaggerAPIProfileVOResponse "获取用户资料成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户资料不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/profiles/{userID} [get]
func (ctrl *UserProfileController) GetProfileByUserIDHandler(c *gin.Context) {
	const operation = "UserProfileController.GetProfileByUserIDHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("获取用户资料请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}
	// 可以在此添加权限校验逻辑：是否是用户本人或管理员在请求？当前依赖于路由注册时的中间件。

	// 2. 调用服务层获取资料。
	profileVO, err := ctrl.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "用户资料不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误或未预期的服务层错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功获取用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess(c, profileVO, "获取资料成功")
}

// UpdateProfileHandler 处理更新用户个人资料的请求。
// @Summary 更新用户资料
// @Description 用户或管理员更新指定用户ID的个人资料信息（如昵称、头像、地区等）。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Param userID path string true "要更新资料的用户ID"
// @Param body body dto.UpdateProfileDTO true "包含待更新字段的资料信息"
// @Success 200 {object} docs.SwaggerAPIProfileVOResponse "资料更新成功，返回更新后的资料信息"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、用户ID为空)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户资料不存在"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/profiles/{userID} [put]
func (ctrl *UserProfileController) UpdateProfileHandler(c *gin.Context) {
	const operation = "UserProfileController.UpdateProfileHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("更新用户资料请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}
	// 权限校验：是否是用户本人或管理员？依赖中间件。

	// 2. 绑定并校验请求体数据。
	var updateProfileDTO dto.UpdateProfileDTO
	if err := c.ShouldBindJSON(&updateProfileDTO); err != nil {
		ctrl.logger.Warn("更新用户资料请求参数绑定失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 3. 调用服务层执行更新逻辑。
	profileVO, err := ctrl.profileService.UpdateProfile(c.Request.Context(), userID, &updateProfileDTO)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要更新的用户资料不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 返回成功响应。
	ctrl.logger.Info("成功更新用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess(c, profileVO, "资料更新成功")
}

// DeleteProfileHandler 处理删除用户个人资料的请求。
// @Summary 删除用户资料
// @Description 用户或管理员删除指定用户ID的个人资料记录。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Param userID path string true "要删除资料的用户ID"
// @Success 200 {object} docs.SwaggerAPIEmptyResponse "资料删除成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如用户ID为空)"
// @Failure 404 {object} docs.SwaggerAPIErrorResponseString "指定的用户资料不存在 (如果服务层认为删除不存在的记录是错误)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/profiles/{userID} [delete]
func (ctrl *UserProfileController) DeleteProfileHandler(c *gin.Context) {
	const operation = "UserProfileController.DeleteProfileHandler"

	// 1. 获取并校验路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("删除用户资料请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}
	// 权限校验：是否是用户本人或管理员？依赖中间件。

	// 2. 调用服务层执行删除逻辑。
	err := ctrl.profileService.DeleteProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要删除的用户资料不存在" { // 假设服务层对删除不存在记录返回此业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误 (较少见)
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功删除用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess[vo.Empty](c, vo.Empty{}, "资料删除成功") // 使用 vo.Empty
}

// RegisterRoutes 注册与用户资料管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 将此控制器的所有API端点集中定义和注册。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
func (ctrl *UserProfileController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /profiles 子路由组，用于管理用户资料资源
	profileRoutes := group.Group("/profiles")
	{
		// 创建用户资料
		// 场景：用户首次填写资料，或系统自动创建。
		// 预期权限：已认证的用户（通常只能创建自己的资料，需在服务层或网关校验UserID）或管理员。
		profileRoutes.POST("", ctrl.CreateProfileHandler) // 移除中间件

		// 获取指定用户的资料
		// 场景：用户查看自己或他人（如果允许）的资料，管理员查看用户资料。
		// 预期权限：已认证的用户（获取自己的资料）或管理员（获取任何用户的资料）。需进行UserID匹配或角色判断。
		profileRoutes.GET("/:userID", ctrl.GetProfileByUserIDHandler) // 移除中间件

		// 更新指定用户的资料
		// 场景：用户修改自己的资料，管理员修改用户资料。
		// 预期权限：已认证的用户（更新自己的资料）或管理员（更新任何用户的资料）。需进行UserID匹配或角色判断。
		profileRoutes.PUT("/:userID", ctrl.UpdateProfileHandler) // 移除中间件

		// 删除指定用户的资料
		// 场景：用户注销账户时可能触发，或管理员操作。
		// 预期权限：已认证的用户（删除自己的资料）或管理员（删除任何用户的资料）。需进行UserID匹配或角色判断。
		profileRoutes.DELETE("/:userID", ctrl.DeleteProfileHandler) // 移除中间件
	}
}
