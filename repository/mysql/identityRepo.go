package mysql

import (
	"context"
	"errors"
	"user_hub/models/entities"
	"user_hub/models/enums"
	"user_hub/userError"

	"gorm.io/gorm"
)

// IdentityCredential 定义身份验证所需的最小字段集结构体
// - 用于返回用户身份凭证的核心信息
type IdentityCredential struct {
	UserID     string `gorm:"column:user_id"`    // 用户 ID
	Credential string `gorm:"column:credential"` // 身份凭证（如密码哈希）
}

// IdentityRepository 定义用户身份仓库接口
// - 提供管理用户身份的 CRUD 操作
type IdentityRepository interface {
	// CreateIdentity 创建新的用户身份
	// - 输入: ctx 上下文, identity 用户身份实体
	// - 输出: error 操作错误
	CreateIdentity(ctx context.Context, identity *entities.UserIdentity) error

	// GetIdentityByID 根据身份 ID 获取完整的用户身份信息
	// - 输入: ctx 上下文, identityID 身份 ID
	// - 输出: *entities.UserIdentity 用户身份实体指针, error 操作错误
	GetIdentityByID(ctx context.Context, identityID uint) (*entities.UserIdentity, error)

	// GetIdentityByTypeAndIdentifier 根据身份类型和标识符获取用户凭证
	// - 输入: ctx 上下文, identityType 身份类型, identifier 身份标识符
	// - 输出: IdentityCredential 用户凭证指针, error 操作错误
	GetIdentityByTypeAndIdentifier(ctx context.Context, identityType enums.IdentityType, identifier string) (*IdentityCredential, error)

	// UpdateIdentity 更新用户身份信息
	// - 输入: ctx 上下文, identity 用户身份实体
	// - 输出: error 操作错误
	UpdateIdentity(ctx context.Context, identity *entities.UserIdentity) error

	// DeleteIdentity 删除用户身份
	// - 输入: ctx 上下文, identityID 身份 ID
	// - 输出: error 操作错误
	DeleteIdentity(ctx context.Context, identityID uint) error

	// GetIdentitiesByUserID 根据用户 ID 获取所有相关身份
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: []*entities.UserIdentity 用户身份列表, error 操作错误
	GetIdentitiesByUserID(ctx context.Context, userID string) ([]*entities.UserIdentity, error)

	// GetIdentityTypesByUserID 根据用户 ID 获取所有相关身份类型
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: []enums.IdentityType 用户身份类型列表, error 操作错误
	GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error)
}

// identityRepository 实现 IdentityRepository 接口的结构体
type identityRepository struct {
	db *gorm.DB // GORM 数据库实例
}

// NewIdentityRepository 创建 IdentityRepository 实例
// - 输入: db GORM 数据库实例
// - 输出: IdentityRepository 接口实现
func NewIdentityRepository(db *gorm.DB) IdentityRepository {
	return &identityRepository{db: db}
}

// CreateIdentity 创建用户身份
// - 输入: ctx 上下文, identity 用户身份实体
// - 输出: error 操作错误
// - SQL: INSERT INTO user_identities (identity_id, user_id, identity_type, identifier, credential, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)
func (r *identityRepository) CreateIdentity(ctx context.Context, identity *entities.UserIdentity) error {
	// 使用 GORM 创建用户身份记录
	return r.db.WithContext(ctx).Create(identity).Error
}

// GetIdentityByID 根据身份 ID 获取完整的用户身份信息
// - 输入: ctx 上下文, identityID 身份 ID
// - 输出: *entities.UserIdentity 用户身份实体指针, error 操作错误
// - SQL: SELECT * FROM user_identities WHERE identity_id = ? LIMIT 1
func (r *identityRepository) GetIdentityByID(ctx context.Context, identityID uint) (*entities.UserIdentity, error) {
	// 1. 查询用户身份
	// - 从 user_identities 表中查找指定 identity_id 的记录
	var identity entities.UserIdentity
	err := r.db.WithContext(ctx).
		Where("identity_id = ?", identityID).
		First(&identity).Error

	// 2. 处理查询结果
	// - 如果记录不存在，返回自定义错误 ErrIdentityNotFound
	// - 如果发生其他错误，返回原始错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userError.ErrIdentityNotFound
		}
		return nil, err
	}
	return &identity, nil
}

// GetIdentityByTypeAndIdentifier 根据身份类型和标识符获取用户凭证
// - 输入: ctx 上下文, identityType 身份类型, identifier 身份标识符
// - 输出: *IdentityCredential 用户凭证指针, error 操作错误
// - SQL: SELECT user_id, credential FROM user_identities WHERE identity_type = ? AND identifier = ? LIMIT 1
func (r *identityRepository) GetIdentityByTypeAndIdentifier(ctx context.Context, identityType enums.IdentityType, identifier string) (*IdentityCredential, error) {
	// 1. 查询用户凭证
	// - 从 user_identities 表中选择 user_id 和 credential 字段
	// - 根据 identity_type 和 identifier 筛选记录
	var cred IdentityCredential
	err := r.db.WithContext(ctx).
		Select("user_id, credential").
		Table("user_identities").
		Where("identity_type = ? AND identifier = ?", identityType, identifier).
		First(&cred).Error

	// 2. 处理查询结果
	// - 如果记录不存在，返回自定义错误 ErrIdentityNotFound
	// - 如果发生其他错误，返回原始错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userError.ErrIdentityNotFound
		}
		return nil, err
	}
	return &cred, nil
}

// UpdateIdentity 更新用户身份信息
// - 输入: ctx 上下文, identity 用户身份实体
// - 输出: error 操作错误
// - SQL: UPDATE user_identities SET user_id = ?, identity_type = ?, identifier = ?, credential = ?, updated_at = ? WHERE identity_id = ?
func (r *identityRepository) UpdateIdentity(ctx context.Context, identity *entities.UserIdentity) error {
	// 使用 GORM 更新用户身份记录
	return r.db.WithContext(ctx).Save(identity).Error
}

// DeleteIdentity 删除用户身份
// - 输入: ctx 上下文, identityID 身份 ID
// - 输出: error 操作错误
// - SQL: DELETE FROM user_identities WHERE identity_id = ?
func (r *identityRepository) DeleteIdentity(ctx context.Context, identityID uint) error {
	// 使用 GORM 删除指定 identity_id 的用户身份记录
	return r.db.WithContext(ctx).Where("identity_id = ?", identityID).Delete(&entities.UserIdentity{}).Error
}

// GetIdentitiesByUserID 根据用户 ID 获取所有相关身份
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: []*entities.UserIdentity 用户身份列表, error 操作错误
// - SQL: SELECT * FROM user_identities WHERE user_id = ?
func (r *identityRepository) GetIdentitiesByUserID(ctx context.Context, userID string) ([]*entities.UserIdentity, error) {
	// 1. 查询用户身份列表
	// - 从 user_identities 表中查找所有 user_id 匹配的记录
	var identities []*entities.UserIdentity
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&identities).Error

	// 2. 处理查询结果
	// - 如果发生错误，返回错误
	// - 成功时返回身份列表
	if err != nil {
		return nil, err
	}
	return identities, nil
}

// GetIdentityTypesByUserID 根据用户 ID 获取所有相关身份类型
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: []enums.IdentityType 用户身份类型列表, error 操作错误
// - SQL: SELECT identity_type FROM user_identities WHERE user_id = ?
func (r *identityRepository) GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error) {
	// 1. 查询用户身份类型
	// - 从 user_identities 表中选择 identity_type 字段
	var identityTypes []enums.IdentityType
	err := r.db.WithContext(ctx).
		Table("user_identities").
		Where("user_id = ?", userID).
		Pluck("identity_type", &identityTypes).Error

	// 2. 处理查询结果
	// - 如果发生错误，返回错误
	// - 成功时返回身份类型列表
	if err != nil {
		return nil, err
	}
	return identityTypes, nil
}
