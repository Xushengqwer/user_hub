package mysql

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装
	"github.com/Xushengqwer/go-common/commonerrors"

	// 导入公共模块的 enums
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/user_hub/models/entities"

	"gorm.io/gorm"
)

// UserRepository 定义了与核心用户（User）数据存储相关的操作接口。
// - 它抽象了数据库交互，提供用户的 CRUD（创建、读取、更新、删除）以及状态管理功能。
type UserRepository interface {
	// CreateUser 持久化一个新的核心用户记录。
	// - 接收应用上下文和待创建的用户实体。
	// - 如果数据库操作失败，则返回包装后的错误。
	CreateUser(ctx context.Context, db *gorm.DB, user *entities.User) error

	// GetUserByID 根据用户 ID 检索单个核心用户的完整信息。
	// - 如果未找到匹配的用户，将返回 commonerrors.ErrRepoNotFound。
	// - 其他数据库错误将被包装后返回。
	GetUserByID(ctx context.Context, userID string) (*entities.User, error)

	// UpdateUser 更新一个已存在的核心用户信息。
	// - 注意：此方法当前使用 GORM 的 Updates，通常只更新非零值字段。服务层应确保传入的实体是期望的状态，或考虑使用 Select 指定更新字段。
	// - 如果数据库操作失败，则返回包装后的错误。
	UpdateUser(ctx context.Context, user *entities.User) error

	// DeleteUser 根据用户 ID（软）删除一个核心用户记录。
	// - GORM 的 Delete 默认执行软删除（如果模型包含 gorm.DeletedAt）。
	// - 如果数据库操作失败，则返回包装后的错误。
	DeleteUser(ctx context.Context, db *gorm.DB, userID string) error

	// BlackUser 将指定用户 ID 的状态更新为“拉黑”。
	// - 直接更新 status 字段。
	// - 如果数据库操作失败，则返回包装后的错误。
	BlackUser(ctx context.Context, userID string) error
}

// userRepository 是 UserRepository 接口基于 GORM 的实现。
type userRepository struct {
	db *gorm.DB // db 是 GORM 数据库连接实例
}

// NewUserRepository 创建一个新的 userRepository 实例。
// - 依赖注入 GORM 数据库连接。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser 实现接口方法，持久化用户记录。
func (r *userRepository) CreateUser(ctx context.Context, db *gorm.DB, user *entities.User) error {
	// 执行数据库创建操作
	if err := db.WithContext(ctx).Create(user).Error; err != nil {
		// 包装创建操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("userRepo.CreateUser: 创建用户失败: %w", err)
	}
	// 操作成功，返回 nil
	return nil
}

// GetUserByID 实现接口方法，根据 ID 获取用户信息。
func (r *userRepository) GetUserByID(ctx context.Context, userID string) (*entities.User, error) {
	var user entities.User
	// 执行数据库查询操作
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error

	if err != nil {
		// 检查是否是 GORM 的“记录未找到”错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 根据约定，记录未找到时返回统一的公共错误
			return nil, commonerrors.ErrRepoNotFound // 使用 commonerrors 包的错误
		}
		// 包装其他查询错误，添加中文上下文信息
		return nil, fmt.Errorf("userRepo.GetUserByID: 查询用户失败 (UserID: %s): %w", userID, err)
	}
	// 查询成功，返回找到的用户实体和 nil 错误
	return &user, nil
}

// UpdateUser 实现接口方法，更新用户信息。
func (r *userRepository) UpdateUser(ctx context.Context, user *entities.User) error {
	// 使用 GORM 的 Updates 方法更新用户记录，通常只更新非零值字段。
	// Model(&entities.User{UserID: userManage.UserID}) 指定了更新条件基于主键。
	// Updates(userManage) 传入包含待更新字段的实体。
	result := r.db.WithContext(ctx).Model(&entities.User{UserID: user.UserID}).Updates(user)
	if result.Error != nil {
		// 包装更新操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("userRepo.UpdateUser: 更新用户信息失败 (UserID: %s): %w", user.UserID, result.Error)
	}
	// 可选：检查 result.RowsAffected == 0，如果需要区分“未找到可更新的”和“成功更新”
	// if result.RowsAffected == 0 {
	//     return commonerrors.ErrRepoNotFound // 如果需要严格区分
	// }
	// 操作成功，返回 nil
	return nil
}

// DeleteUser 实现接口方法，删除用户。
// - 使用传入的 db 对象执行操作，使其能够参与外部事务。
func (r *userRepository) DeleteUser(ctx context.Context, db *gorm.DB, userID string) error {
	// GORM 的 Delete 需要一个模型实例来确定表名
	// 使用传入的 db (可能是事务 tx，也可能是原始连接)
	result := db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.User{})
	if result.Error != nil {
		// 包装删除操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("userRepo.DeleteUser: 删除用户失败 (UserID: %s): %w", userID, result.Error)
	}
	// 对于软删除，即使记录不存在，GORM 通常也不会返回 ErrRecordNotFound，
	// 而是 RowsAffected 为 0 且 error 为 nil。
	// 如果需要严格区分“未找到可删除的记录”，可以检查 result.RowsAffected。
	// if result.RowsAffected == 0 {
	//     // 注意：如果 DeleteUser 之前没有 GetUserByID 检查，这里返回 ErrRepoNotFound 可能更合适。
	//     // 但如果 Service 层已确认用户存在，则 RowsAffected 为 0 可能表示其他并发问题或 GORM 的特定行为。
	//     // 服务层通常会先 Get 再 Delete，所以这里不返回 ErrRepoNotFound 也是常见的。
	// }
	return nil
}

// BlackUser 实现接口方法，设置用户为黑名单状态。
func (r *userRepository) BlackUser(ctx context.Context, userID string) error {
	// 使用 GORM 的 Update 方法更新单个字段 'status'
	result := r.db.WithContext(ctx).Model(&entities.User{}).Where("user_id = ?", userID).Update("status", enums.StatusBlacklisted)
	if result.Error != nil {
		// 包装更新状态操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("userRepo.BlackUser: 拉黑用户失败 (UserID: %s): %w", userID, result.Error)
	}
	// 可选：检查 result.RowsAffected == 0，如果需要区分“用户未找到”和“成功拉黑”
	// if result.RowsAffected == 0 {
	//     return commonerrors.ErrRepoNotFound // 如果需要严格区分
	// }
	// 操作成功，返回 nil
	return nil
}
