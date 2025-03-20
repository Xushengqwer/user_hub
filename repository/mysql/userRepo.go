package mysql

import (
	"context"
	"errors"
	"user_hub/models/entities"
	"user_hub/models/enums"
	"user_hub/userError"

	"gorm.io/gorm"
)

// UserRepository 定义用户仓库接口
// - 提供管理用户的 CRUD 操作及黑名单设置
type UserRepository interface {
	// CreateUser 创建新用户
	// - 输入: ctx 上下文, user 用户实体
	// - 输出: error 操作错误
	CreateUser(ctx context.Context, user *entities.User) error

	// GetUserByID 根据用户 ID 获取用户信息
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *entities.User 用户实体指针, error 操作错误
	GetUserByID(ctx context.Context, userID string) (*entities.User, error)

	// UpdateUser 更新用户信息
	// - 输入: ctx 上下文, user 用户实体
	// - 输出: error 操作错误
	UpdateUser(ctx context.Context, user *entities.User) error

	// DeleteUser 删除用户
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: error 操作错误
	DeleteUser(ctx context.Context, userID string) error

	// BlackUser 设置用户黑名单状态
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: error 操作错误
	BlackUser(ctx context.Context, userID string) error
}

// userRepository 实现 UserRepository 接口的结构体
type userRepository struct {
	db *gorm.DB // GORM 数据库实例
}

// NewUserRepository 创建 UserRepository 实例
// - 输入: db GORM 数据库实例
// - 输出: UserRepository 接口实现
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser 创建用户
// - 输入: ctx 上下文, user 用户实体
// - 输出: error 操作错误
// - SQL: INSERT INTO users (user_id, ...) VALUES (?, ...)
func (r *userRepository) CreateUser(ctx context.Context, user *entities.User) error {
	// 使用 GORM 创建用户记录
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID 根据用户 ID 获取用户
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: *entities.User 用户实体指针, error 操作错误
// - SQL: SELECT * FROM users WHERE user_id = ? LIMIT 1
func (r *userRepository) GetUserByID(ctx context.Context, userID string) (*entities.User, error) {
	// 1. 查询用户信息
	// - 从 users 表中查找指定 user_id 的记录
	var user entities.User
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error

	// 2. 处理查询结果
	// - 如果记录不存在，返回自定义错误 ErrUserNotFound
	// - 如果发生其他错误，返回原始错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userError.ErrUserNotFound
		}
		return nil, err // 其他数据库错误原样返回
	}
	return &user, nil
}

// UpdateUser 更新用户信息
// - 输入: ctx 上下文, user 用户实体
// - 输出: error 操作错误
// - SQL: UPDATE users SET ... WHERE user_id = ?
func (r *userRepository) UpdateUser(ctx context.Context, user *entities.User) error {
	// 使用 GORM 的 Updates 方法更新用户记录
	// - 只更新非零值字段，避免覆盖数据库中的零值
	return r.db.WithContext(ctx).Model(&entities.User{UserID: user.UserID}).Updates(user).Error
}

// DeleteUser 删除用户
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
// - SQL: DELETE FROM users WHERE user_id = ?
func (r *userRepository) DeleteUser(ctx context.Context, userID string) error {
	// 使用 GORM 删除指定 user_id 的用户记录
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.User{}).Error
}

// BlackUser 设置用户黑名单状态
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
// - SQL: UPDATE users SET status = ? WHERE user_id = ?
func (r *userRepository) BlackUser(ctx context.Context, userID string) error {
	// 使用 GORM 更新用户状态为黑名单
	// - 将 status 字段设置为 enums.Blacklisted
	return r.db.WithContext(ctx).Model(&entities.User{}).Where("user_id = ?", userID).Update("status", enums.Blacklisted).Error
}
