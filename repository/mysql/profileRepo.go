package mysql

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装
	"github.com/Xushengqwer/go-common/commonerrors"

	"github.com/Xushengqwer/user_hub/models/entities"

	"gorm.io/gorm"
)

// ProfileRepository 定义了与用户资料（UserProfile）数据存储相关的操作接口。
// - 它抽象了数据库交互，为用户资料提供 CRUD（创建、读取、更新、删除）功能。
type ProfileRepository interface {
	// CreateProfile 持久化一条新的用户资料记录。
	// - 接收应用上下文和待创建的用户资料实体。
	// - 如果数据库操作失败，则返回包装后的错误。
	CreateProfile(ctx context.Context, db *gorm.DB, profile *entities.UserProfile) error

	// GetProfileByUserID 根据用户 ID 检索单个用户资料的完整信息。
	// - 如果未找到匹配的用户资料，将返回 commonerrors.ErrRepoNotFound。
	// - 其他数据库错误将被包装后返回。
	GetProfileByUserID(ctx context.Context, userID string) (*entities.UserProfile, error)

	// UpdateProfile 更新一个已存在的用户资料信息。
	// - 注意：此方法当前使用 GORM 的 Save，会更新记录的所有字段。服务层应确保传入的实体是期望的完整状态。
	// - 如果数据库操作失败，则返回包装后的错误。
	UpdateProfile(ctx context.Context, profile *entities.UserProfile) error

	// DeleteProfile 根据用户 ID 删除一条用户资料记录。
	// - 如果数据库操作失败，则返回包装后的错误。
	DeleteProfile(ctx context.Context, db *gorm.DB, userID string) error
}

// profileRepository 是 ProfileRepository 接口基于 GORM 的实现。
type profileRepository struct {
	db *gorm.DB // db 是 GORM 数据库连接实例
}

// NewProfileRepository 创建一个新的 profileRepository 实例。
// - 依赖注入 GORM 数据库连接。
func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

// CreateProfile 实现接口方法，持久化用户资料记录。
func (r *profileRepository) CreateProfile(ctx context.Context, db *gorm.DB, profile *entities.UserProfile) error {
	// 执行数据库创建操作，使用传入的 db 对象 (可以是原始连接或事务)
	if err := db.WithContext(ctx).Create(profile).Error; err != nil {
		// 包装创建操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("profileRepo.CreateProfile: 创建用户资料失败 (UserID: %s): %w", profile.UserID, err)
	}
	// 操作成功，返回 nil
	return nil
}

// GetProfileByUserID 实现接口方法，根据用户 ID 获取用户资料。
func (r *profileRepository) GetProfileByUserID(ctx context.Context, userID string) (*entities.UserProfile, error) {
	var profile entities.UserProfile
	// 执行数据库查询操作
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error

	if err != nil {
		// 检查是否是 GORM 的“记录未找到”错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 根据约定，记录未找到时返回统一的公共错误
			return nil, commonerrors.ErrRepoNotFound
		}
		// 包装其他查询错误，添加中文上下文信息
		return nil, fmt.Errorf("profileRepo.GetProfileByUserID: 查询用户资料失败 (UserID: %s): %w", userID, err)
	}
	// 查询成功，返回找到的用户资料实体和 nil 错误
	return &profile, nil
}

// UpdateProfile 实现接口方法，更新用户资料信息。
func (r *profileRepository) UpdateProfile(ctx context.Context, profile *entities.UserProfile) error {
	// 注意：Save 会更新记录的所有字段。服务层应确保传入的 profile 实体是期望的完整状态，
	// 否则未在 profile 中设置的字段在数据库中可能会被更新为零值。
	// 如果仅需更新部分字段，服务层应先获取完整实体，修改后再调用此方法，
	// 或者此方法内部改为使用 Updates 配合 Select 来精确控制更新字段。
	if err := r.db.WithContext(ctx).Save(profile).Error; err != nil {
		// 包装更新操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("profileRepo.UpdateProfile: 更新用户资料失败 (UserID: %s): %w", profile.UserID, err)
	}
	// 操作成功，返回 nil
	return nil
}

// DeleteProfile 实现接口方法，删除用户资料。
// - 使用传入的 db 对象执行操作，使其能够参与外部事务。
func (r *profileRepository) DeleteProfile(ctx context.Context, db *gorm.DB, userID string) error {
	// GORM 的 Delete 需要一个模型实例（即使是空实例）来确定表名
	// 使用传入的 db (可能是事务 tx，也可能是原始连接)
	result := db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.UserProfile{})
	if result.Error != nil {
		// 包装删除操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("profileRepo.DeleteProfile: 删除用户资料失败 (UserID: %s): %w", userID, result.Error)
	}
	// 如果没有行受影响，可能意味着该用户的资料本就不存在。
	// 服务层通常会先判断用户是否存在，或者删除操作本身具有幂等性。
	// if result.RowsAffected == 0 {
	//     return commonerrors.ErrRepoNotFound // 如果需要严格区分“未找到可删除的记录”
	// }
	return nil
}
