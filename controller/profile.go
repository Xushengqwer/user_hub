package controller

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net/http"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/constants"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/dto"
	service "github.com/Xushengqwer/user_hub/service/profile"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// UserProfileController 处理与用户详细资料（Profile）相关的 HTTP 请求。
// 依赖于 service.UserProfileService 来执行核心业务逻辑。
type UserProfileController struct {
	profileService service.UserProfileService     // profileService: 用户资料管理服务的实例。
	jwtUtil        dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger         *core.ZapLogger                // logger: 日志记录器。
	db             *gorm.DB                       // <-- 新增：数据库连接实例
}

// NewUserProfileController 创建一个新的 UserProfileController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - profileService: 实现了 service.UserProfileService 接口的服务实例。
//   - jwtUtil: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *UserProfileController: 初始化完成的控制器实例。
func NewUserProfileController(
	profileService service.UserProfileService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
	db *gorm.DB, // <-- 新增：接收 *gorm.DB 参数
) *UserProfileController {
	return &UserProfileController{
		profileService: profileService,
		jwtUtil:        jwtUtil,
		logger:         logger, // 存储 logger
		db:             db,     // <-- 存储数据库连接实例
	}
}

// UpdateProfileHandler 处理当前认证用户更新自己资料的请求。
// @Summary 更新我的用户资料
// @Description 当前认证用户更新自己的个人资料信息（如昵称、性别、地区等）。头像更新请使用专门的头像上传接口。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Param body body dto.UpdateProfileDTO true "包含待更新字段的资料信息（不含头像URL）"
// @Success 200 {object} docs.SwaggerAPIProfileVOResponse "资料更新成功，返回更新后的资料信息"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误)"
// @Failure 401 {object} docs.SwaggerAPIErrorResponseString "未授权或认证失败"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败或用户资料不存在)"
// @Router /api/v1/user-hub/profile [put]
func (ctrl *UserProfileController) UpdateProfileHandler(c *gin.Context) {
	const operation = "UserProfileController.UpdateProfileHandler"

	userIDRaw, exists := c.Get(string(constants.UserIDKey))
	if !exists {
		ctrl.logger.Error("无法从上下文中获取UserID用于更新资料", zap.String("operation", operation))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户未认证")
		return
	}
	userID, ok := userIDRaw.(string)
	if !ok || userID == "" {
		ctrl.logger.Error("从上下文中获取的UserID无效用于更新资料", zap.String("operation", operation), zap.Any("rawUserID", userIDRaw))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户认证信息无效")
		return
	}

	var updateProfileDTO dto.UpdateProfileDTO
	if err := c.ShouldBindJSON(&updateProfileDTO); err != nil {
		ctrl.logger.Warn("更新用户资料请求参数绑定失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效: "+err.Error())
		return
	}

	profileVO, err := ctrl.profileService.UpdateProfile(c.Request.Context(), userID, &updateProfileDTO)
	if err != nil {
		// 根据您的要求，如果服务层返回 "要更新的用户资料不存在"，则视为服务器内部错误
		if err.Error() == "要更新的用户资料不存在" || err.Error() == "无效的性别值" { // 也处理服务层可能返回的性别校验错误
			ctrl.logger.Error("更新用户资料时发生内部错误或数据校验问题",
				zap.String("operation", operation),
				zap.String("userID", userID),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "更新用户资料时发生内部错误")
		} else if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else { // 其他不太可能发生的业务错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	ctrl.logger.Info("成功更新用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess(c, profileVO, "资料更新成功")
}

// UploadAvatarHandler 处理用户头像上传的请求。
// @Summary 上传我的头像
// @Description 当前认证用户上传自己的头像文件。成功后返回新的头像URL。
// @Tags 资料管理 (Profile Management)
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "头像文件 (multipart/form-data key: 'avatar')"
// @Success 200 {object} response.APIResponse[map[string]string] "头像上传成功，返回包含新头像URL的map"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求无效 (如文件过大、类型不支持、未提供文件)"
// @Failure 401 {object} docs.SwaggerAPIErrorResponseString "未授权或认证失败"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如文件上传到COS失败、数据库更新失败)"
// @Router /api/v1/user-hub/profile/avatar [post]
func (ctrl *UserProfileController) UploadAvatarHandler(c *gin.Context) {
	const operation = "UserProfileController.UploadAvatarHandler"

	userIDRaw, exists := c.Get(string(constants.UserIDKey))
	if !exists {
		ctrl.logger.Error("无法从上下文中获取UserID用于头像上传", zap.String("operation", operation))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户未认证")
		return
	}
	userID, ok := userIDRaw.(string)
	if !ok || userID == "" {
		ctrl.logger.Error("从上下文中获取的UserID无效用于头像上传", zap.String("operation", operation), zap.Any("rawUserID", userIDRaw))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户认证信息无效")
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		ctrl.logger.Warn("获取上传文件失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "无法读取上传的文件: "+err.Error())
		return
	}
	defer file.Close()

	const maxFileSize = 5 * 1024 * 1024 // 5MB
	if header.Size > maxFileSize {
		ctrl.logger.Warn("上传文件过大", zap.String("operation", operation), zap.String("userID", userID), zap.Int64("fileSize", header.Size))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, fmt.Sprintf("文件大小不能超过 %dMB", maxFileSize/1024/1024))
		return
	}

	newAvatarURL, err := ctrl.profileService.UploadAndSetAvatar(c.Request.Context(), userID, header.Filename, file, header.Size)
	if err != nil {
		// 根据服务层返回的错误类型进行处理
		// 假设 ErrCodeThirdPartyServiceError = 50004
		if errors.Is(err, commonerrors.ErrThirdPartyServiceError) { // 检查是否为第三方服务错误
			ctrl.logger.Error("服务层报告腾讯云COS服务失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
			response.RespondError(c, http.StatusBadGateway, response.ErrCodeThirdPartyServiceError, "头像上传服务暂时不可用，请稍后重试") // 使用 502 Bad Gateway 可能更合适
		} else if errors.Is(err, commonerrors.ErrSystemError) { // 其他系统内部错误（例如服务层返回 "用户不存在或用户资料未初始化"，但我们已将其归为内部错误）
			ctrl.logger.Error("服务层报告系统内部错误", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "上传头像失败，请稍后重试")
		}
		return
	}

	ctrl.logger.Info("头像上传并设置成功",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.String("newAvatarURL", newAvatarURL),
	)
	response.RespondSuccess(c, map[string]string{"avatar_url": newAvatarURL}, "头像上传成功")
}

// GetMyProfileHandler 处理当前认证用户获取自己账户聚合信息的请求。
// @Summary 获取我的账户详情 (核心信息 + 资料)
// @Description 获取当前认证用户的核心账户信息（如角色、状态）和详细个人资料（如昵称、头像）。
// @Tags 资料管理 (Profile Management)
// @Accept json
// @Produce json
// @Success 200 {object} docs.SwaggerAPIMyAccountDetailResponse "获取账户详情成功"
// @Failure 401 {object} docs.SwaggerAPIErrorResponseString "未授权或认证失败"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据不一致或数据库查询失败)"
// @Router /api/v1/user-hub/profile [get]
func (ctrl *UserProfileController) GetMyProfileHandler(c *gin.Context) {
	const operation = "UserProfileController.GetMyProfileHandler"

	userIDRaw, exists := c.Get("UserID")
	if !exists {
		ctrl.logger.Error("无法从上下文中获取UserID", zap.String("operation", operation))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户未认证")
		return
	}
	userID, ok := userIDRaw.(string)
	if !ok || userID == "" {
		ctrl.logger.Error("从上下文中获取的UserID无效", zap.String("operation", operation), zap.Any("rawUserID", userIDRaw))
		response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "用户认证信息无效")
		return
	}

	// 调用服务层获取聚合的账户详情
	accountDetailVO, err := ctrl.profileService.GetMyAccountDetail(c.Request.Context(), userID)
	if err != nil {
		// 服务层的 GetMyAccountDetail 在找不到核心用户或profile时，会返回包装了 ErrSystemError 的错误
		// 因此这里主要判断是否为 ErrSystemError
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("服务层获取账户详情失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
			// 根据错误信息，可以判断是核心用户问题还是profile问题，但统一返回内部错误
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "获取账户信息时发生内部错误")
		} else {
			// 其他不太可能发生的业务错误（服务层应已处理为系统错误）
			ctrl.logger.Error("服务层获取账户详情返回未知错误", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "获取账户信息失败")
		}
		return
	}

	ctrl.logger.Info("成功获取当前用户账户详情",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	response.RespondSuccess(c, accountDetailVO, "获取账户详情成功")
}

// RegisterRoutes 注册与用户资料管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 将此控制器的所有API端点集中定义和注册。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
//     例如，如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/profiles" 子分组的完整基础路径将是 "/user-hub/api/v1/profiles"。
func (ctrl *UserProfileController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /profiles 子路由组，用于管理用户资料资源
	profileRoutes := group.Group("/profile")
	{
		// 更新指定用户的资料
		// 场景：用户（包括普通用户和管理员）修改自己的资料
		profileRoutes.PUT("", ctrl.UpdateProfileHandler)

		// 用户上传自己的头像
		// 场景：包含用户和管理员都可以
		profileRoutes.POST("/avatar", ctrl.UploadAvatarHandler) // 上传我的头像

		// 处理当前认证用户获取自己账户聚合信息的请求
		// 场景： 前端需要使用这个加载用户头像，个人信息
		profileRoutes.GET("", ctrl.GetMyProfileHandler) // 修改为调用 GetMyProfileHandler
	}
}
