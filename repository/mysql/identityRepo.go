package mysql

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装
	"github.com/Xushengqwer/go-common/commonerrors"

	// 假设 IdentityCredential 移到了 dto 包
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	"github.com/Xushengqwer/user_hub/models/enums"

	"gorm.io/gorm"
)

// IdentityRepository 定义了与用户身份（UserIdentity）数据存储相关的操作接口。
// - 它抽象了数据库交互的细节，允许服务层以统一的方式访问和管理用户身份数据。
type IdentityRepository interface {
	// CreateIdentity 持久化一个新的用户身份记录。
	// - 接收应用上下文和待创建的用户身份实体。
	// - 如果数据库操作失败，则返回包装后的错误。
	CreateIdentity(ctx context.Context, db *gorm.DB, identity *entities.UserIdentity) error

	// GetIdentityByID 根据主键 ID 检索单个用户身份的完整信息。
	// - 如果未找到匹配的身份，将返回 commonerrors.ErrRepoNotFound。
	// - 其他数据库错误将被包装后返回。
	GetIdentityByID(ctx context.Context, identityID uint) (*entities.UserIdentity, error)

	// GetIdentityByTypeAndIdentifier 根据身份类型和唯一标识符检索用户的核心凭证信息。
	// - 主要用于登录验证等场景，只选择必要字段以提高效率。
	// - 如果未找到匹配的凭证，将返回 commonerrors.ErrRepoNotFound。
	// - 其他数据库错误将被包装后返回。
	GetIdentityByTypeAndIdentifier(ctx context.Context, identityType enums.IdentityType, identifier string) (*dto.IdentityCredential, error)

	// UpdateIdentity 更新一个已存在的用户身份记录。
	// - 注意：此方法当前使用 GORM 的 Save，会更新所有字段。服务层应确保传入的实体是期望的状态。
	// - 如果数据库操作失败，则返回包装后的错误。
	UpdateIdentity(ctx context.Context, identity *entities.UserIdentity) error

	// DeleteIdentity 根据主键 ID 删除一个用户身份记录。
	// - 如果数据库操作失败，则返回包装后的错误。
	DeleteIdentity(ctx context.Context, db *gorm.DB, identityID uint) error

	// GetIdentitiesByUserID 检索指定用户 ID 关联的所有身份记录。
	// - 如果用户没有任何身份记录，将返回一个空列表和 nil 错误。
	// - 如果数据库查询失败，则返回包装后的错误。
	GetIdentitiesByUserID(ctx context.Context, userID string) ([]*entities.UserIdentity, error)

	// GetIdentityTypesByUserID 检索指定用户 ID 所拥有的所有身份类型。
	// - 使用 Pluck 高效获取单列数据。
	// - 如果用户没有任何身份记录，将返回一个空列表和 nil 错误。
	// - 如果数据库查询失败，则返回包装后的错误。
	GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error)

	// DeleteIdentitiesByUserID 根据用户 ID （软）删除该用户的所有身份记录。
	// 设计目的:
	//  - 在用户注销或被管理员删除时，级联删除其所有登录凭证。
	//  - 确保操作的原子性（当在事务中调用时）。
	// 参数:
	//  - db: 用于执行此操作的 GORM 数据库句柄 (可以是原始连接或事务对象)。
	//  - ctx: 请求上下文。
	//  - userID: 要删除其所有身份的用户 ID。
	// 返回:
	//  - error: 如果数据库操作失败，则返回包装后的错误。如果用户没有任何身份记录，不视为错误。
	DeleteIdentitiesByUserID(ctx context.Context, db *gorm.DB, userID string) error
}

// identityRepository 是 IdentityRepository 接口基于 GORM 的实现。
type identityRepository struct {
	db *gorm.DB // db 是 GORM 数据库连接实例
}

// NewIdentityRepository 创建一个新的 identityRepository 实例。
// - 依赖注入 GORM 数据库连接。
func NewIdentityRepository(db *gorm.DB) IdentityRepository {
	return &identityRepository{db: db}
}

// CreateIdentity 实现接口方法，持久化用户身份记录。
func (r *identityRepository) CreateIdentity(ctx context.Context, db *gorm.DB, identity *entities.UserIdentity) error {
	// 执行数据库创建操作
	if err := db.WithContext(ctx).Create(identity).Error; err != nil {
		// 包装创建操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("identityRepo.CreateIdentity: 创建身份失败: %w", err)
	}
	// 操作成功，返回 nil
	return nil
}

// GetIdentityByID 实现接口方法，根据 ID 获取身份信息。
func (r *identityRepository) GetIdentityByID(ctx context.Context, identityID uint) (*entities.UserIdentity, error) {
	var identity entities.UserIdentity
	// 执行数据库查询操作
	err := r.db.WithContext(ctx).
		Where("identity_id = ?", identityID).
		First(&identity).Error

	if err != nil {
		// 检查是否是 GORM 的“记录未找到”错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 根据约定，记录未找到时返回统一的公共错误
			return nil, commonerrors.ErrRepoNotFound // 使用 commonerrors 包的错误
		}
		// 包装其他查询错误，添加中文上下文信息
		return nil, fmt.Errorf("identityRepo.GetIdentityByID: 查询身份失败 (ID: %d): %w", identityID, err)
	}
	// 查询成功，返回找到的身份实体和 nil 错误
	return &identity, nil
}

// GetIdentityByTypeAndIdentifier 实现接口方法，根据类型和标识符获取凭证。
func (r *identityRepository) GetIdentityByTypeAndIdentifier(ctx context.Context, identityType enums.IdentityType, identifier string) (*dto.IdentityCredential, error) {
	var cred dto.IdentityCredential // 使用 dto 包下的结构体
	// 执行数据库查询操作，只选择需要的字段
	err := r.db.WithContext(ctx).
		Select("user_id, credential").
		Table("user_identities"). // 明确指定表名，因为 DTO 通常不是 GORM 模型
		Where("identity_type = ? AND identifier = ?", identityType, identifier).
		First(&cred).Error

	if err != nil {
		// 检查是否是 GORM 的“记录未找到”错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录未找到时返回统一的公共错误
			return nil, commonerrors.ErrRepoNotFound // 使用 commonerrors 包的错误
		}
		// 包装其他查询错误，添加中文上下文信息
		return nil, fmt.Errorf("identityRepo.GetIdentityByTypeAndIdentifier: 查询凭证失败 (类型: %d, 标识符: %s): %w", identityType, identifier, err)
	}
	// 查询成功，返回凭证 DTO 和 nil 错误
	return &cred, nil
}

// UpdateIdentity 实现接口方法，更新用户身份信息。
func (r *identityRepository) UpdateIdentity(ctx context.Context, identity *entities.UserIdentity) error {
	// 注意：Save 会更新所有字段。确保调用方传入的是完整的、期望状态的实体。
	// 执行数据库更新操作
	if err := r.db.WithContext(ctx).Save(identity).Error; err != nil {
		// 包装更新操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("identityRepo.UpdateIdentity: 更新身份失败 (ID: %d): %w", identity.IdentityID, err)
	}
	// 操作成功，返回 nil
	return nil
}

// DeleteIdentity 实现接口方法，删除用户身份。
// - 使用传入的 db 对象执行操作，使其能够参与外部事务。
func (r *identityRepository) DeleteIdentity(ctx context.Context, db *gorm.DB, identityID uint) error {
	// GORM 的 Delete 需要一个模型实例来确定表名
	// 使用传入的 db (可能是事务 tx，也可能是原始连接)
	result := db.WithContext(ctx).Where("identity_id = ?", identityID).Delete(&entities.UserIdentity{})
	if result.Error != nil {
		// 包装删除操作时发生的错误，添加中文上下文信息
		return fmt.Errorf("identityRepo.DeleteIdentity: 删除身份失败 (ID: %d): %w", identityID, result.Error)
	}
	// if result.RowsAffected == 0 {
	//     return commonerrors.ErrRepoNotFound // 如果需要严格区分“未找到”和“删除成功”
	// }
	return nil
}

// GetIdentitiesByUserID 实现接口方法，获取用户的所有身份。
func (r *identityRepository) GetIdentitiesByUserID(ctx context.Context, userID string) ([]*entities.UserIdentity, error) {
	var identities []*entities.UserIdentity
	// Find 操作在未找到记录时，返回空 slice 和 nil error，这是 GORM 的正常行为。
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&identities).Error
	if err != nil {
		// 包装查询列表时发生的错误，添加中文上下文信息
		return nil, fmt.Errorf("identityRepo.GetIdentitiesByUserID: 查询用户身份列表失败 (UserID: %s): %w", userID, err)
	}
	// 查询成功（即使结果为空列表），返回身份列表和 nil 错误
	return identities, nil
}

// GetIdentityTypesByUserID 实现接口方法，获取用户的所有身份类型。
func (r *identityRepository) GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error) {
	var identityTypes []enums.IdentityType
	// Pluck 操作在未找到记录时，返回空 slice 和 nil error。
	err := r.db.WithContext(ctx).
		Table("user_identities"). // Pluck 需要明确表名
		Where("user_id = ?", userID).
		Pluck("identity_type", &identityTypes).Error

	if err != nil {
		// 包装 Pluck 操作时发生的错误，添加中文上下文信息
		return nil, fmt.Errorf("identityRepo.GetIdentityTypesByUserID: 获取用户身份类型失败 (UserID: %s): %w", userID, err)
	}
	// 查询成功（即使结果为空列表），返回身份类型列表和 nil 错误
	return identityTypes, nil
}

// DeleteIdentitiesByUserID 实现接口方法，根据用户 ID （软）删除该用户的所有身份记录。
// - 使用传入的 db 对象执行操作，使其能够参与外部事务。
func (r *identityRepository) DeleteIdentitiesByUserID(ctx context.Context, db *gorm.DB, userID string) error {
	// 对于软删除模型 (UserIdentity 包含 gorm.DeletedAt)，GORM 的 Delete 会更新 deleted_at 字段。
	// Where 条件会匹配所有 user_id 为指定值的记录。
	result := db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entities.UserIdentity{})
	if result.Error != nil {
		return fmt.Errorf("identityRepo.DeleteIdentitiesByUserID: 删除用户的所有身份记录失败 (UserID: %s): %w", userID, result.Error)
	}
	// 即使没有记录被删除 (result.RowsAffected == 0)，也不应视为错误，
	// 因为目标是确保该用户最终没有活动的身份记录。
	// 例如，如果一个用户没有任何身份信息，调用此方法删除其身份是正常的，不应报错。
	return nil
}
