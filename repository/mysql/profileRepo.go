package mysql

import (
	"context"
	"errors"
	"user_hub/models/entities"
	"user_hub/userError"

	"gorm.io/gorm"
)

// ProfileRepository 定义用户资料仓库接口
// - 提供管理用户资料的 CRUD 操作
type ProfileRepository interface {
	// CreateProfile 创建新的用户资料
	// - 输入: ctx 上下文, profile 用户资料实体
	// - 输出: error 操作错误
	CreateProfile(ctx context.Context, profile *entities.UserProfile) error

	// GetProfileByUserID 根据用户 ID 获取用户资料
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *entities.UserProfile 用户资料指针, error 操作错误
	GetProfileByUserID(ctx context.Context, userID string) (*entities.UserProfile, error)

	// UpdateProfile 更新用户资料信息
	// - 输入: ctx 上下文, profile 用户资料实体
	// - 输出: error 操作错误
	UpdateProfile(ctx context.Context, profile *entities.UserProfile) error

	// DeleteProfile 删除用户资料
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: error 操作错误
	DeleteProfile(ctx context.Context, userID string) error
}

// profileRepository 实现 ProfileRepository 接口的结构体
type profileRepository struct {
	db *gorm.DB // GORM 数据库实例
}

// NewProfileRepository 创建 ProfileRepository 实例
// - 输入: db GORM 数据库实例
// - 输出: ProfileRepository 接口实现
func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

// CreateProfile 创建用户资料
// - 输入: ctx 上下文, profile 用户资料实体
// - 输出: error 操作错误
// - SQL: INSERT INTO user_profiles (user_id, ...) VALUES (?, ...)
func (r *profileRepository) CreateProfile(ctx context.Context, profile *entities.UserProfile) error {
	// 使用 GORM 创建用户资料记录
	return r.db.WithContext(ctx).Create(profile).Error
}

// GetProfileByUserID 根据用户 ID 获取用户资料
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: *entities.UserProfile 用户资料指针, error 操作错误
// - SQL: SELECT * FROM user_profiles WHERE user_id = ? LIMIT 1
func (r *profileRepository) GetProfileByUserID(ctx context.Context, userID string) (*entities.UserProfile, error) {
	// 1. 查询用户资料
	// - 从 user_profiles 表中查找指定 user_id 的记录
	var profile entities.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error

	// 2. 处理查询结果
	// - 如果记录不存在，返回自定义错误 ErrProfileNotFound
	// - 如果发生其他错误，返回原始错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userError.ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateProfile 更新用户资料
// - 输入: ctx 上下文, profile 用户资料实体
// - 输出: error 操作错误
// - SQL: UPDATE user_profiles SET ... WHERE user_id = ?
func (r *profileRepository) UpdateProfile(ctx context.Context, profile *entities.UserProfile) error {
	// 使用 GORM 更新用户资料记录
	return r.db.WithContext(ctx).Save(profile).Error
}

// DeleteProfile 删除用户资料
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
// - SQL: DELETE FROM user_profiles WHERE user_id = ?
func (r *profileRepository) DeleteProfile(ctx context.Context, userID string) error {
	// 使用 GORM 删除指定 user_id 的用户资料记录
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.UserProfile{}).Error
}
