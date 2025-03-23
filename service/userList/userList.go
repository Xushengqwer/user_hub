package userList

import (
	"context"
	"user_hub/models/dto"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
)

// QueryService 定义查询服务接口
type QueryService interface {
	// ListUsersWithProfile 分页查询用户及其资料信息
	// - 使用场景: 管理员查看用户列表及其资料，支持过滤、排序和分页
	// - 输入: ctx 上下文, dto 查询条件 DTO
	// - 输出: []*vo.UserWithProfileVO 用户资料列表, int64 总记录数, error 操作错误
	ListUsersWithProfile(ctx context.Context, dto *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error)
}

// queryService 实现 QueryService 接口的结构体
type queryService struct {
	repo mysql.JoinQuery // 联合查询仓库实例
}

// NewQueryService 创建 QueryService 实例
// - 使用场景: 初始化查询服务
// - 输入: repo 联合查询仓库实例
// - 输出: QueryService 接口实现
func NewQueryService(repo mysql.JoinQuery) QueryService {
	return &queryService{repo: repo}
}

// ListUsersWithProfile 分页查询用户及其资料信息
// - 使用场景: 管理员查看用户列表及其资料，支持过滤、排序和分页
// - 输入: ctx 上下文, dto 查询条件 DTO
// - 输出: []*vo.UserWithProfileVO 用户资料列表, int64 总记录数, error 操作错误
func (s *queryService) ListUsersWithProfile(ctx context.Context, dto *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error) {
	// 1. 构造查询条件
	// - 将 DTO 转换为仓库层所需的 UserQuery 结构体
	query := mysql.UserQuery{
		Filters:          dto.Filters,
		LikeFilters:      dto.LikeFilters,
		TimeRangeFilters: dto.TimeRangeFilters,
		OrderBy:          dto.OrderBy,
		Page:             dto.Page,
		PageSize:         dto.PageSize,
	}

	// 2. 调用仓库层执行联合查询
	// - 查询用户及其资料信息，返回结果和总数
	results, total, err := s.repo.ListUsersWithProfile(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	// 3. 构造用户资料视图对象列表
	// - 将仓库层返回的实体转换为 VO
	var userWithProfileVOs []*vo.UserWithProfileVO
	for _, result := range results {
		voItem := &vo.UserWithProfileVO{
			UserID:    result.UserID,
			Role:      result.Role,
			Status:    result.Status,
			Nickname:  result.Nickname,
			AvatarURL: result.AvatarURL,
			Gender:    result.Gender,
			Province:  result.Province,
			City:      result.City,
		}
		userWithProfileVOs = append(userWithProfileVOs, voItem)
	}

	// 4. 返回查询结果和总数
	return userWithProfileVOs, total, nil
}

// todo  漏掉了两个字段-- 创建时间和删除时间
