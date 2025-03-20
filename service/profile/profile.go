package profile

import (
	"context"
	"errors"
	"user_hub/models/dto"
	"user_hub/models/entities"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/userError"
)

// ProfileService 定义资料管理服务接口
type ProfileService interface {
	// CreateProfile 创建用户资料
	// - 使用场景: 用户首次填写个人资料
	// - 输入: ctx 上下文, dto 创建资料 DTO
	// - 输出: *vo.ProfileVO 资料响应, error 操作错误
	CreateProfile(ctx context.Context, dto *dto.CreateProfileDTO) (*vo.ProfileVO, error)

	// GetProfileByUserID 根据用户 ID 获取资料
	// - 使用场景: 用户或管理员查看用户资料
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *vo.ProfileVO 资料响应, error 操作错误
	GetProfileByUserID(ctx context.Context, userID string) (*vo.ProfileVO, error)

	// UpdateProfile 更新用户资料
	// - 使用场景: 用户或管理员更新用户资料
	// - 输入: ctx 上下文, userID 用户 ID, dto 更新资料 DTO
	// - 输出: *vo.ProfileVO 资料响应, error 操作错误
	UpdateProfile(ctx context.Context, userID string, dto *dto.UpdateProfileDTO) (*vo.ProfileVO, error)

	// DeleteProfile 删除用户资料
	// - 使用场景: 用户或管理员删除用户资料
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: error 操作错误
	DeleteProfile(ctx context.Context, userID string) error
}

// profileService 实现 ProfileService 接口的结构体
type profileService struct {
	repo mysql.ProfileRepository // 资料仓库实例
}

// NewProfileService 创建 ProfileService 实例
// - 使用场景: 初始化资料管理服务
// - 输入: repo 资料仓库实例
// - 输出: ProfileService 接口实现
func NewProfileService(repo mysql.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

// CreateProfile 创建用户资料
// - 使用场景: 用户首次填写个人资料
// - 输入: ctx 上下文, dto 创建资料 DTO
// - 输出: *vo.ProfileVO 资料响应, error 操作错误
func (s *profileService) CreateProfile(ctx context.Context, dto *dto.CreateProfileDTO) (*vo.ProfileVO, error) {
	// 1. 创建资料实体
	// - 根据 DTO 初始化资料字段
	profile := &entities.UserProfile{
		UserID:    dto.UserID,
		Nickname:  dto.Nickname,
		AvatarURL: dto.AvatarURL,
		Gender:    dto.Gender,
		Province:  dto.Province,
		City:      dto.City,
	}

	// 2. 调用仓库层创建资料
	// - 将资料实体插入数据库
	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, err
	}

	// 3. 构造并返回资料视图对象
	// - 将实体转换为 VO 返回
	profileVO := &vo.ProfileVO{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		AvatarURL: profile.AvatarURL,
		Gender:    profile.Gender,
		Province:  profile.Province,
		City:      profile.City,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
	return profileVO, nil
}

// GetProfileByUserID 根据用户 ID 获取资料
// - 使用场景: 用户或管理员查看用户资料
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: *vo.ProfileVO 资料响应, error 操作错误
func (s *profileService) GetProfileByUserID(ctx context.Context, userID string) (*vo.ProfileVO, error) {
	// 1. 调用仓库层获取资料
	// - 根据用户 ID 查询资料实体
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, userError.ErrProfileNotFound) {
			return nil, err // 返回 nil 和错误，提示“用户资料为空”
		}
		return nil, err
	}

	// 2. 构造并返回资料视图对象
	// - 将实体转换为 VO 返回
	profileVO := &vo.ProfileVO{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		AvatarURL: profile.AvatarURL,
		Gender:    profile.Gender,
		Province:  profile.Province,
		City:      profile.City,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
	return profileVO, nil
}

// UpdateProfile 更新用户资料
// - 使用场景: 用户或管理员更新用户资料
// - 输入: ctx 上下文, userID 用户 ID, dto 更新资料 DTO
// - 输出: *vo.ProfileVO 资料响应, error 操作错误
func (s *profileService) UpdateProfile(ctx context.Context, userID string, dto *dto.UpdateProfileDTO) (*vo.ProfileVO, error) {
	// 1. 查询资料是否存在
	// - 根据用户 ID 获取现有资料实体
	profile, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, userError.ErrProfileNotFound) {
			return nil, err
		}
		return nil, err
	}

	// 2. 更新资料字段
	// - 根据 DTO 更新非空字段
	if dto.Nickname != "" {
		profile.Nickname = dto.Nickname
	}
	if dto.AvatarURL != "" {
		profile.AvatarURL = dto.AvatarURL
	}

	if dto.Gender >= 0 {
		profile.Gender = dto.Gender
	}
	if dto.Province != "" {
		profile.Province = dto.Province
	}
	if dto.City != "" {
		profile.City = dto.City
	}

	// 3. 调用仓库层更新资料
	// - 将更新后的实体保存到数据库
	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return nil, err
	}

	// 4. 构造并返回资料视图对象
	// - 将更新后的实体转换为 VO 返回
	profileVO := &vo.ProfileVO{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		AvatarURL: profile.AvatarURL,
		Gender:    profile.Gender,
		Province:  profile.Province,
		City:      profile.City,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
	return profileVO, nil
}

// DeleteProfile 删除用户资料
// - 使用场景: 用户或管理员删除用户资料
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
func (s *profileService) DeleteProfile(ctx context.Context, userID string) error {
	// 1. 调用仓库层删除资料
	// - 根据用户 ID 删除资料记录
	if err := s.repo.DeleteProfile(ctx, userID); err != nil {
		return err
	}
	return nil
}
