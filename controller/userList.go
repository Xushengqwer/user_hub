package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	"user_hub/models/vo"
	service "user_hub/service/userList"
)

// QueryController 查询控制器
type QueryController struct {
	queryService service.QueryService // 查询服务实例
}

// NewQueryController 创建 QueryController 实例
// - 输入: queryService 查询服务实例
// - 输出: *QueryController 控制器实例
func NewQueryController(queryService service.QueryService) *QueryController {
	return &QueryController{queryService: queryService}
}

// ListUsersWithProfileHandler 处理分页查询用户及其资料请求
// @Summary 分页查询用户及其资料
// @Description 管理员查看用户列表及其资料，支持过滤、排序和分页
// @Tags 查询管理
// @Accept json
// @Produce json
// @Param body body dto.UserQueryDTO true "查询条件"
// @Success 200 {object} response.APIResponse[struct{Users []*vo.UserWithProfileVO;Total int64}] "查询成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "查询失败"
// @Router /users/query [post]
func (ctrl *QueryController) ListUsersWithProfileHandler(c *gin.Context) {
	// 1. 绑定请求数据
	var dto dto.UserQueryDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 调用服务层执行查询
	users, total, err := ctrl.queryService.ListUsersWithProfile(c.Request.Context(), &dto)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "查询失败")
		return
	}

	// 3. 构造响应数据
	// - 将用户列表和总数封装到结构体中
	responseData := struct {
		Users []*vo.UserWithProfileVO `json:"users"`
		Total int64                   `json:"total"`
	}{
		Users: users,
		Total: total,
	}

	// 4. 返回成功响应
	response.RespondSuccess(c, responseData, "查询成功")
}
