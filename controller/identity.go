package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	service "github.com/Xushengqwer/user_hub/service/identity"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// IdentityController 处理与用户多种身份凭证管理相关的 HTTP 请求。
// 例如：用户绑定新的登录方式、修改密码、解绑登录方式等。
type IdentityController struct {
	identityService service.UserIdentityService    // identityService: 用户身份管理服务的实例。
	jwtUtil         dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger          *core.ZapLogger                // logger: 日志记录器。
}

// NewIdentityController 创建一个新的 IdentityController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - identityService: 实现了 service.UserIdentityService 接口的服务实例。
//   - jwtUtil: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *IdentityController: 初始化完成的控制器实例。
func NewIdentityController(
	identityService service.UserIdentityService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) *IdentityController {
	return &IdentityController{
		identityService: identityService,
		jwtUtil:         jwtUtil,
		logger:          logger, // 存储 logger
	}
}

// CreateIdentityHandler 处理为用户创建新身份（绑定新登录方式）的请求。
// @Summary 创建新身份
// @Description 用户或管理员为指定用户绑定一种新的登录方式（如新的账号密码、关联社交账号等）。
// @Tags 身份管理 (Identity Management)
// @Accept json
// @Produce json
// @Param body body dto.CreateIdentityDTO true "创建身份请求的详细信息，包括用户ID、身份类型、标识符和凭证"
// @Success 200 {object} response.APIResponse[vo.IdentityVO] "身份创建成功，返回新创建的身份信息"
// @Failure 400 {object} response.APIResponse[string] "请求参数无效 (如JSON格式错误、必填项缺失) 或 业务逻辑错误 (如身份标识已存在)"
// @Failure 500 {object} response.APIResponse[string] "系统内部错误 (如数据库操作失败、密码加密失败)"
// @Router /api/v1/user-hub/identities [post] // <--- 已更新路径
func (ctrl *IdentityController) CreateIdentityHandler(c *gin.Context) {
	const operation = "IdentityController.CreateIdentityHandler"

	// 1. 绑定并校验请求体数据。
	var createIdentityDTO dto.CreateIdentityDTO
	if err := c.ShouldBindJSON(&createIdentityDTO); err != nil {
		ctrl.logger.Warn("创建新身份请求参数绑定失败", zap.String("operation", operation), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 2. 调用服务层执行创建身份的逻辑。
	identityVO, err := ctrl.identityService.CreateIdentity(c.Request.Context(), &createIdentityDTO)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			// 服务层已记录详细错误，此处返回通用系统错误提示。
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他错误视为业务逻辑错误（例如，身份标识已存在）。
			// 服务层应返回对用户友好的错误信息。
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功创建用户身份",
		zap.String("operation", operation),
		zap.Uint("identityID", identityVO.IdentityID), // 假设 vo.IdentityVO 有 IdentityID
		zap.String("userID", identityVO.UserID),
	)
	response.RespondSuccess(c, identityVO, "身份创建成功")
}

// UpdateIdentityHandler 处理更新用户身份信息（通常是凭证，如密码）的请求。
// @Summary 更新身份信息
// @Description 用户或管理员修改指定身份ID的凭证信息（例如，重置密码）。
// @Tags 身份管理 (Identity Management)
// @Accept json
// @Produce json
// @Param identityID path uint true "要更新的身份记录的唯一ID" Format(uint)
// @Param body body dto.UpdateIdentityDTO true "更新身份请求的详细信息，主要包含新的凭证"
// @Success 200 {object} response.APIResponse[vo.IdentityVO] "身份信息更新成功，返回更新后的身份信息"
// @Failure 400 {object} response.APIResponse[string] "请求参数无效 (如JSON格式错误、身份ID格式无效、新凭证无效)"
// @Failure 404 {object} response.APIResponse[string] "指定的身份记录不存在"
// @Failure 500 {object} response.APIResponse[string] "系统内部错误 (如数据库操作失败、密码加密失败)"
// @Router /api/v1/user-hub/identities/{identityID} [put] // <--- 已更新路径
func (ctrl *IdentityController) UpdateIdentityHandler(c *gin.Context) {
	const operation = "IdentityController.UpdateIdentityHandler"

	// 1. 获取并校验路径参数 identityID。
	identityIDStr := c.Param("identityID")
	identityID, err := strconv.ParseUint(identityIDStr, 10, 64)
	if err != nil {
		ctrl.logger.Warn("更新身份信息请求的 identityID 格式无效",
			zap.String("operation", operation),
			zap.String("identityIDStr", identityIDStr),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "身份 ID 格式无效")
		return
	}

	// 2. 绑定并校验请求体数据。
	var updateIdentityDTO dto.UpdateIdentityDTO
	if err := c.ShouldBindJSON(&updateIdentityDTO); err != nil {
		ctrl.logger.Warn("更新身份信息请求参数绑定失败",
			zap.String("operation", operation),
			zap.Uint64("identityID", identityID),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "请求数据无效")
		return
	}

	// 3. 调用服务层执行更新身份的逻辑。
	identityVO, err := ctrl.identityService.UpdateIdentity(c.Request.Context(), uint(identityID), &updateIdentityDTO)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要更新的身份记录不存在" { // 假设服务层对未找到情况返回此特定业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误（例如，新凭证不符合要求等）
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 返回成功响应。
	ctrl.logger.Info("成功更新用户身份信息",
		zap.String("operation", operation),
		zap.Uint64("identityID", identityID),
	)
	response.RespondSuccess(c, identityVO, "身份信息更新成功")
}

// DeleteIdentityHandler 处理删除用户某个特定身份的请求。
// @Summary 删除身份
// @Description 用户或管理员注销或移除某个特定的登录方式（身份记录）。
// @Tags 身份管理 (Identity Management)
// @Accept json
// @Produce json
// @Param identityID path uint true "要删除的身份记录的唯一ID" Format(uint)
// @Success 200 {object} response.APIResponse[vo.Empty] "身份删除成功"
// @Failure 400 {object} response.APIResponse[string] "请求参数无效 (如身份ID格式无效)"
// @Failure 404 {object} response.APIResponse[string] "指定的身份记录不存在 (如果服务层认为删除不存在的记录是错误)"
// @Failure 500 {object} response.APIResponse[string] "系统内部错误 (如数据库操作失败)"
// @Router /api/v1/user-hub/identities/{identityID} [delete] // <--- 已更新路径
func (ctrl *IdentityController) DeleteIdentityHandler(c *gin.Context) {
	const operation = "IdentityController.DeleteIdentityHandler"

	// 1. 获取并校验路径参数 identityID。
	identityIDStr := c.Param("identityID")
	identityID, err := strconv.ParseUint(identityIDStr, 10, 64)
	if err != nil {
		ctrl.logger.Warn("删除身份请求的 identityID 格式无效",
			zap.String("operation", operation),
			zap.String("identityIDStr", identityIDStr),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "身份 ID 格式无效")
		return
	}

	// 2. 调用服务层执行删除身份的逻辑。
	err = ctrl.identityService.DeleteIdentity(c.Request.Context(), uint(identityID))
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "要删除的身份记录不存在" { // 假设服务层对删除不存在记录返回此业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他视为业务逻辑错误 (虽然删除操作通常业务错误较少，除非有前置条件检查)
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应。
	ctrl.logger.Info("成功删除用户身份",
		zap.String("operation", operation),
		zap.Uint64("identityID", identityID),
	)
	response.RespondSuccess[interface{}](c, nil, "身份删除成功")
}

// GetIdentitiesByUserIDHandler 处理根据用户ID获取其所有身份信息的请求。
// @Summary 获取用户的所有身份信息
// @Description 管理员或用户本人查看指定用户ID关联的所有登录方式/身份凭证信息（不含敏感凭证内容）。
// @Tags 身份管理 (Identity Management)
// @Accept json
// @Produce json
// @Param userID path string true "要查询的用户ID"
// @Success 200 {object} response.APIResponse[vo.IdentityList] "获取用户身份列表成功"
// @Failure 400 {object} response.APIResponse[string] "请求参数无效 (如用户ID为空)"
// @Failure 404 {object} response.APIResponse[string] "指定的用户不存在 (如果服务层检查用户存在性)"
// @Failure 500 {object} response.APIResponse[string] "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/user-hub/users/{userID}/identities [get] // <--- 已更新路径
func (ctrl *IdentityController) GetIdentitiesByUserIDHandler(c *gin.Context) {
	const operation = "IdentityController.GetIdentitiesByUserIDHandler"

	// 1. 获取路径参数 userID。
	userID := c.Param("userID")
	if userID == "" { // 基本校验
		ctrl.logger.Warn("获取用户身份列表请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取身份列表。
	identitiesVO, err := ctrl.identityService.GetIdentitiesByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "用户不存在" { // 假设服务层可能返回此业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			// 其他错误视为业务逻辑错误或未预期的服务层错误
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 构造响应数据并返回。
	//    即使列表为空 (identitiesVO 切片长度为0)，也应返回成功和空列表，而不是错误。
	ctrl.logger.Info("成功获取用户身份列表",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.Int("count", len(identitiesVO)),
	)
	response.RespondSuccess(c, vo.IdentityList{Items: identitiesVO}, "获取用户身份列表成功")
}

// GetIdentityTypesByUserIDHandler 处理根据用户ID获取其所有身份类型的请求。
// @Summary 获取用户的所有身份类型
// @Description 用户或系统查看指定用户ID已绑定的所有登录方式的类型列表。
// @Tags 身份管理 (Identity Management)
// @Accept json
// @Produce json
// @Param userID path string true "要查询的用户ID"
// @Success 200 {object} response.APIResponse[vo.IdentityTypeList] "获取用户身份类型列表成功"
// @Failure 400 {object} response.APIResponse[string] "请求参数无效 (如用户ID为空)"
// @Failure 404 {object} response.APIResponse[string] "指定的用户不存在 (如果服务层检查用户存在性)"
// @Failure 500 {object} response.APIResponse[string] "系统内部错误 (如数据库查询失败)"
// @Router /api/v1/user-hub/users/{userID}/identity-types [get] // <--- 已更新路径
func (ctrl *IdentityController) GetIdentityTypesByUserIDHandler(c *gin.Context) {
	const operation = "IdentityController.GetIdentityTypesByUserIDHandler"

	// 1. 获取路径参数 userID。
	userID := c.Param("userID")
	if userID == "" {
		ctrl.logger.Warn("获取用户身份类型列表请求的用户ID为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "用户 ID 不能为空")
		return
	}

	// 2. 调用服务层获取身份类型列表。
	identityTypes, err := ctrl.identityService.GetIdentityTypesByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrSystemError) {
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else if err.Error() == "用户不存在" { // 假设服务层可能返回此业务错误
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, err.Error())
		} else {
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 构造响应数据并返回。
	ctrl.logger.Info("成功获取用户身份类型列表",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.Int("count", len(identityTypes)),
	)
	response.RespondSuccess(c, vo.IdentityTypeList{Items: identityTypes}, "获取用户身份类型列表成功")
}

// RegisterRoutes 注册与用户身份管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 将此控制器的所有API端点集中定义和注册。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
//     例如，如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/identities" 子分组的完整基础路径将是 "/user-hub/api/v1/identities"，
//     而 "/users" 子分组的完整基础路径将是 "/user-hub/api/v1/users"。
func (ctrl *IdentityController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /identities 子路由组，用于管理单个身份记录的 CRUD 操作。
	identitiesRoutes := group.Group("/identities")
	{
		// 创建新身份 (例如，用户绑定新登录方式)
		// 预期需要认证, 允许用户和管理员操作 (由网关处理认证和基础角色判断)
		identitiesRoutes.POST("", ctrl.CreateIdentityHandler) // 完整路径: /user-hub/api/v1/identities

		// 更新身份信息 (例如，修改密码)
		// 预期需要认证，允许管理员或用户本人操作 (网关处理认证，服务层或后续逻辑需处理本人或管理员判断)
		identitiesRoutes.PUT("/:identityID", ctrl.UpdateIdentityHandler) // 完整路径: /user-hub/api/v1/identities/:identityID

		// 删除身份 (例如，用户解绑登录方式)
		// 预期需要认证，允许管理员或用户本人操作 (同上)
		identitiesRoutes.DELETE("/:identityID", ctrl.DeleteIdentityHandler) // 完整路径: /user-hub/api/v1/identities/:identityID
	}

	// 创建 /users 子路由组下的身份相关查询接口。
	// 这些接口通常用于查询某个用户关联的身份信息。
	userSpecificIdentityRoutes := group.Group("/users")
	{
		// 获取指定用户的所有身份类型
		// 预期需要认证，允许管理员或用户本人操作 (同上)
		// 完整路径: /user-hub/api/v1/users/:userID/identity-types
		userSpecificIdentityRoutes.GET("/:userID/identity-types", ctrl.GetIdentityTypesByUserIDHandler)

		// 获取指定用户的所有身份信息（例如，查看其所有注册的账号密码）
		// 预期需要认证，仅允许管理员操作 (网关处理认证，服务层或后续逻辑需处理本人或管理员判断)
		// 完整路径: /user-hub/api/v1/users/:userID/identities
		userSpecificIdentityRoutes.GET("/:userID/identities", ctrl.GetIdentitiesByUserIDHandler)
	}
}
