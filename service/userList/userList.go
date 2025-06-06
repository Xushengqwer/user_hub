package userList

import (
	"context"
	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"go.uber.org/zap"                       // 引入 zap 用于日志字段

	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
)

// UserListQueryService 定义了用户列表查询相关的服务接口。
// 设计目的:
// - 提供管理员或特定场景下对用户列表及其关联信息（如Profile）进行复杂分页查询的功能。
// - 封装与仓库层 `JoinQuery` 的交互细节。
// 使用场景:
// - 管理后台的用户管理页面，需要展示用户列表，并支持按条件筛选、排序和分页。
type UserListQueryService interface {
	// ListUsersWithProfile 分页查询用户及其关联的Profile信息。
	// 参数:
	//  - ctx: 请求上下文。
	//  - dto: 包含过滤、排序和分页参数的查询 DTO，直接从 Controller 层传递而来。
	// 返回:
	//  - []*vo.UserWithProfileVO: 用户及其Profile信息的视图对象列表。
	//  - int64: 符合查询条件的总记录数。
	//  - error: 操作过程中发生的任何错误，通常是系统错误。
	ListUsersWithProfile(ctx context.Context, dto *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error)
}

// userListQueryService 是 UserListQueryService 接口的实现。
type userListQueryService struct {
	repo   mysql.JoinQuery // repo: 联合查询仓库，负责执行实际的数据库查询。
	logger *core.ZapLogger // logger: 日志记录器。
	// db *gorm.DB // 对于只读查询服务，db 可能不是必需的，除非 JoinQuery 方法也需要外部事务控制。
}

// NewUserListQueryService 创建一个新的 userListQueryService 实例。
// 设计原因:
// - 依赖注入确保了服务的可测试性和灵活性。
func NewUserListQueryService(
	repo mysql.JoinQuery,
	logger *core.ZapLogger, // 注入 logger
	// db *gorm.DB, // 如果需要，也注入 db
) UserListQueryService { // 返回接口类型
	return &userListQueryService{ // 返回结构体指针
		repo:   repo,
		logger: logger,
		// db: db,
	}
}

// ListUsersWithProfile 实现接口方法，执行用户列表的分页条件查询。
func (s *userListQueryService) ListUsersWithProfile(ctx context.Context, dto *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error) {
	const operation = "UserListQueryService.ListUsersWithProfile"
	s.logger.Info("开始查询用户列表及其Profile信息",
		zap.String("operation", operation),
		zap.Any("queryDTO", dto), // 记录查询参数，注意敏感信息处理（如果DTO中包含）
	)

	// 1. 直接调用仓库层的 ListUsersWithProfile 方法。
	//    - 我们之前已重构仓库层，使其直接接收 dto.UserQueryDTO 并返回 []*vo.UserWithProfileVO。
	//    - 因此，服务层不再需要进行 DTO 到仓库查询结构体的转换，也不再需要手动将仓库结果映射到 VO。
	results, total, err := s.repo.ListUsersWithProfile(ctx, dto)
	if err != nil {
		s.logger.Error("调用仓库查询用户列表及其Profile失败",
			zap.String("operation", operation),
			zap.Any("queryDTO", dto),
			zap.Error(err), // 记录从仓库层返回的原始错误
		)
		// 向上层返回通用系统错误
		return nil, 0, commonerrors.ErrSystemError
	}

	s.logger.Info("成功查询用户列表及其Profile信息",
		zap.String("operation", operation),
		zap.Int64("totalRecords", total),
		zap.Int("returnedRecords", len(results)),
	)

	// 2. 直接返回仓库层的结果。
	//    - `CreatedAt` 和 `UpdatedAt` 字段的映射问题：
	//      由于仓库层的 `joinQuery.go` 中的 `Select` 语句已经包含了 `users.created_at` 和 `users.updated_at`，
	//      并且 `vo.UserWithProfileVO` 中也定义了 `CreatedAt` 和 `UpdatedAt` 字段，
	//      GORM 的 `Scan(&results)` (在 `joinQuery.go` 中) 会自动将查询结果映射到 `vo.UserWithProfileVO` 的相应字段。
	//      因此，这里无需手动处理这些字段的映射。
	return results, total, nil
}
