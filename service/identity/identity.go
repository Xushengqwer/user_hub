package service

import (
	"context"
	"errors"
	"user_hub/models/dto"
	"user_hub/models/entities"
	"user_hub/models/enums"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/userError"
)

// IdentityService 定义身份管理服务接口
type IdentityService interface {
	// CreateIdentity 为用户创建新身份
	// - 使用场景: 用户绑定新登录方式（如小程序、手机号）
	// - 输入: ctx 上下文, dto 创建身份 DTO
	// - 输出: *vo.IdentityVO 身份响应, error 操作错误
	CreateIdentity(ctx context.Context, dto *dto.CreateIdentityDTO) (*vo.IdentityVO, error)

	// UpdateIdentity 更新身份信息（如修改密码）
	// - 使用场景: 用户修改密码
	// - 输入: ctx 上下文, identityID 身份 ID, dto 更新身份 DTO
	// - 输出: *vo.IdentityVO 身份响应, error 操作错误
	UpdateIdentity(ctx context.Context, identityID uint, dto *dto.UpdateIdentityDTO) (*vo.IdentityVO, error)

	// DeleteIdentity 删除指定身份
	// - 使用场景: 用户注销某个身份（如小程序身份）
	// - 输入: ctx 上下文, identityID 身份 ID
	// - 输出: error 操作错误
	DeleteIdentity(ctx context.Context, identityID uint) error

	// GetIdentitiesByUserID 获取用户的所有身份信息
	// - 使用场景: 管理员查看用户的所有身份信息
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: []*vo.IdentityVO 身份列表, error 操作错误
	GetIdentitiesByUserID(ctx context.Context, userID string) ([]*vo.IdentityVO, error)

	// GetIdentityTypesByUserID 获取用户绑定的身份类型
	// - 使用场景: 用户查看自己绑定的身份类型
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: []enums.IdentityType 身份类型列表, error 操作错误
	GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error)
}

// identityService 实现 IdentityService 接口的结构体
type identityService struct {
	repo mysql.IdentityRepository // 身份仓库实例
}

// NewIdentityService 创建 IdentityService 实例
// - 使用场景: 初始化身份管理服务
// - 输入: repo 身份仓库实例
// - 输出: IdentityService 接口实现
func NewIdentityService(repo mysql.IdentityRepository) IdentityService {
	return &identityService{repo: repo}
}

// CreateIdentity 为用户创建新身份
// - 使用场景: 用户绑定新登录方式（如小程序、手机号）
// - 输入: ctx 上下文, dto 创建身份 DTO
// - 输出: *vo.IdentityVO 身份响应, error 操作错误
func (s *identityService) CreateIdentity(ctx context.Context, dto *dto.CreateIdentityDTO) (*vo.IdentityVO, error) {
	// 1. 创建身份实体
	identity := &entities.UserIdentity{
		UserID:       dto.UserID,
		IdentityType: dto.IdentityType,
		Identifier:   dto.Identifier,
		Credential:   dto.Credential,
	}

	// 2. 调用仓库层创建身份
	if err := s.repo.CreateIdentity(ctx, identity); err != nil {
		return nil, err
	}

	// 3. 构造并返回身份视图对象
	identityVO := &vo.IdentityVO{
		IdentityID:   identity.IdentityID,
		UserID:       identity.UserID,
		IdentityType: identity.IdentityType,
		Identifier:   identity.Identifier,
		CreatedAt:    identity.CreatedAt,
		UpdatedAt:    identity.UpdatedAt,
	}
	return identityVO, nil
}

// UpdateIdentity 更新身份信息（如修改密码）
// - 使用场景: 用户修改密码
// - 输入: ctx 上下文, identityID 身份 ID, dto 更新身份 DTO
// - 输出: *vo.IdentityVO 身份响应, error 操作错误
func (s *identityService) UpdateIdentity(ctx context.Context, identityID uint, dto *dto.UpdateIdentityDTO) (*vo.IdentityVO, error) {
	// 1. 查询身份是否存在
	identity, err := s.repo.GetIdentityByID(ctx, identityID)
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			return nil, err
		}
		return nil, err
	}

	// 2. 更新凭证
	identity.Credential = dto.Credential

	// 3. 调用仓库层更新身份
	if err := s.repo.UpdateIdentity(ctx, identity); err != nil {
		return nil, err
	}

	// 4. 构造并返回身份视图对象
	identityVO := &vo.IdentityVO{
		IdentityID:   identity.IdentityID,
		UserID:       identity.UserID,
		IdentityType: identity.IdentityType,
		Identifier:   identity.Identifier,
		CreatedAt:    identity.CreatedAt,
		UpdatedAt:    identity.UpdatedAt,
	}
	return identityVO, nil
}

// DeleteIdentity 删除指定身份
// - 使用场景: 用户注销某个身份（如小程序身份）
// - 输入: ctx 上下文, identityID 身份 ID
// - 输出: error 操作错误
func (s *identityService) DeleteIdentity(ctx context.Context, identityID uint) error {
	// 1. 调用仓库层删除身份
	if err := s.repo.DeleteIdentity(ctx, identityID); err != nil {
		return err
	}
	return nil
}

// GetIdentitiesByUserID 获取用户的所有身份信息
// - 使用场景: 管理员查看用户的所有身份信息
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: []*vo.IdentityVO 身份列表, error 操作错误
func (s *identityService) GetIdentitiesByUserID(ctx context.Context, userID string) ([]*vo.IdentityVO, error) {
	// 1. 调用仓库层获取身份列表
	identities, err := s.repo.GetIdentitiesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. 构造身份视图对象列表
	var identityVOs []*vo.IdentityVO
	for _, identity := range identities {
		identityVO := &vo.IdentityVO{
			IdentityID:   identity.IdentityID,
			UserID:       identity.UserID,
			IdentityType: identity.IdentityType,
			Identifier:   identity.Identifier,
			CreatedAt:    identity.CreatedAt,
			UpdatedAt:    identity.UpdatedAt,
		}
		identityVOs = append(identityVOs, identityVO)
	}
	return identityVOs, nil
}

// GetIdentityTypesByUserID 获取用户绑定的身份类型
// - 使用场景: 用户查看自己绑定的身份类型
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: []enums.IdentityType 身份类型列表, error 操作错误
func (s *identityService) GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error) {
	// 1. 调用仓库层获取身份类型列表
	identityTypes, err := s.repo.GetIdentityTypesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return identityTypes, nil
}
