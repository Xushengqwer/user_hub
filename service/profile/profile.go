package profile

import (
	"context"
	"errors"
	"user_hub/models/enums"

	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"go.uber.org/zap"                       // 引入 zap 用于日志字段

	"user_hub/models/dto"
	"user_hub/models/entities"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	// 移除了 user_hub/userError 的导入

	"gorm.io/gorm"
)

// UserProfileService 定义了管理用户详细资料（如昵称、头像、性别、地区等）的服务接口。
// 设计目的:
// - 将用户的基础资料信息与核心用户账户（User）和身份凭证（UserIdentity）分离管理。
// - 提供用户个性化信息的 CRUD 操作。
// 使用场景:
// - 用户在注册后完善个人资料。
// - 用户修改自己的昵称、头像等信息。
// - 系统展示用户的公开资料。
type UserProfileService interface {
	// CreateProfile 为指定用户首次创建其个人资料记录。
	// 使用场景:
	//  - 用户完成基础注册后，引导用户填写更详细的个人信息。
	//  - 系统在某些流程中（如首次社交登录后）自动为用户创建一条空的或部分填充的资料记录。
	// 参数:
	//  - ctx: 请求上下文。
	//  - dto: 包含用户ID及待创建资料信息的 DTO。UserID 必须提供。
	// 返回:
	//  - *vo.ProfileVO: 成功创建的用户资料的视图对象。
	//  - error: 操作过程中发生的任何错误，可能是业务错误或系统错误。
	CreateProfile(ctx context.Context, dto *dto.CreateProfileDTO) (*vo.ProfileVO, error)

	// GetProfileByUserID 根据用户ID检索该用户的详细资料。
	// 使用场景:
	//  - 用户查看或编辑自己的个人资料页面。
	//  - 其他用户查看某用户的公开资料（需注意权限控制和信息脱敏，可能在VO层面处理）。
	// 参数:
	//  - userID: 要查询的用户ID。
	// 返回:
	//  - *vo.ProfileVO: 用户资料的视图对象。如果用户资料不存在，将返回业务错误。
	//  - error: 操作过程中发生的任何错误。
	GetProfileByUserID(ctx context.Context, userID string) (*vo.ProfileVO, error)

	// UpdateProfile 更新指定用户的个人资料信息。
	// 使用场景:
	//  - 用户在个人资料设置页面修改自己的昵称、头像、性别、地区等。
	// 参数:
	//  - userID: 要更新资料的用户ID。
	//  - dto: 包含待更新资料字段的 DTO。DTO中的字段为可选更新，服务会根据DTO中提供的非空/非零值进行更新。
	// 返回:
	//  - *vo.ProfileVO: 更新后的用户资料的视图对象。
	//  - error: 操作过程中发生的任何错误。
	UpdateProfile(ctx context.Context, userID string, dto *dto.UpdateProfileDTO) (*vo.ProfileVO, error)

	// DeleteProfile 删除指定用户的个人资料记录。
	// 使用场景:
	//  - 用户注销账户时，作为清理流程的一部分（如果业务规定需要删除资料）。
	//  - 管理员根据规定移除某个用户的资料信息。
	//  - 注意：通常用户资料与核心用户账户是强关联的，删除资料可能需要谨慎处理或有特定业务含义。
	// 参数:
	//  - userID: 要删除资料的用户ID。
	// 返回:
	//  - error: 操作过程中发生的任何错误。
	DeleteProfile(ctx context.Context, userID string) error
}

// userProfileService 是 UserProfileService 接口的实现。
type userProfileService struct {
	repo   mysql.ProfileRepository // repo: 用户资料数据仓库。
	db     *gorm.DB                // db: GORM数据库连接实例，用于传递给仓库层的写操作方法。
	logger *core.ZapLogger         // logger: 日志记录器。
}

// NewUserProfileService 创建一个新的 userProfileService 实例。
// 设计原因:
// - 依赖注入确保了服务的可测试性和灵活性。
func NewUserProfileService(
	repo mysql.ProfileRepository,
	db *gorm.DB, // 注入 db
	logger *core.ZapLogger, // 注入 logger
) UserProfileService { // 返回接口类型
	return &userProfileService{ // 返回结构体指针
		repo:   repo,
		db:     db,     // 存储 db
		logger: logger, // 存储 logger
	}
}

// profileEntityToVO 是一个内部辅助函数，用于将数据库实体 `entities.UserProfile` 转换为对外暴露的视图对象 `vo.ProfileVO`。
// 设计原因:
// - 视图与模型的解耦。
// - 可以在此进行数据转换或裁剪。
func profileEntityToVO(profile *entities.UserProfile) *vo.ProfileVO {
	if profile == nil {
		return nil
	}
	return &vo.ProfileVO{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		AvatarURL: profile.AvatarURL,
		Gender:    profile.Gender,
		Province:  profile.Province,
		City:      profile.City,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

// CreateProfile 实现接口方法，创建用户资料 (修复时间戳问题)
func (s *userProfileService) CreateProfile(ctx context.Context, dto *dto.CreateProfileDTO) (*vo.ProfileVO, error) {
	const operation = "UserProfileService.CreateProfile"

	// 1. 校验UserID是否已存在对应的Profile (逻辑不变)
	existingProfile, err := s.repo.GetProfileByUserID(ctx, dto.UserID)
	if err == nil && existingProfile != nil {
		s.logger.Warn("尝试为已存在资料的用户创建新资料",
			zap.String("operation", operation),
			zap.String("userID", dto.UserID),
		)
		return nil, errors.New("该用户的资料已存在，请使用更新操作")
	}
	if err != nil && !errors.Is(err, commonerrors.ErrRepoNotFound) {
		s.logger.Error("创建资料前检查用户资料是否存在失败",
			zap.String("operation", operation),
			zap.String("userID", dto.UserID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// 2. 准备实体 (逻辑不变)
	profileEntity := &entities.UserProfile{
		UserID:    dto.UserID,
		Nickname:  dto.Nickname,
		AvatarURL: dto.AvatarURL,
		Gender:    dto.Gender,
		Province:  dto.Province,
		City:      dto.City,
		// 注意：这里不手动设置 CreatedAt 和 UpdatedAt，让数据库默认值生效
	}

	// 3. 调用仓库层创建资料 (逻辑不变)
	if err := s.repo.CreateProfile(ctx, profileEntity); err != nil {
		s.logger.Error("调用仓库创建用户资料失败",
			zap.String("operation", operation),
			zap.String("userID", dto.UserID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// *** 关键修改：创建成功后，重新从数据库获取一次记录 ***
	// 这是为了确保返回给客户端的 VO 中包含数据库实际生成的 CreatedAt 和 UpdatedAt 时间戳，
	// 而不是实体对象中可能仍然是零值的时间。
	createdProfileEntity, err := s.repo.GetProfileByUserID(ctx, dto.UserID)
	if err != nil {
		// 如果刚创建就获取失败，说明可能有严重问题
		s.logger.Error("创建用户资料后获取记录失败",
			zap.String("operation", operation),
			zap.String("userID", dto.UserID),
			zap.Error(err),
		)
		// 虽然创建可能成功了，但无法确认并返回完整数据，也应报告错误
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功创建用户资料",
		zap.String("operation", operation),
		zap.String("userID", createdProfileEntity.UserID),
		zap.Uint("profileID", createdProfileEntity.ID), // 使用从数据库读回的实体 ID
	)

	// 4. 转换并返回从数据库读回的实体对应的 VO
	return profileEntityToVO(createdProfileEntity), nil
}

// GetProfileByUserID 实现接口方法，获取用户资料。
func (s *userProfileService) GetProfileByUserID(ctx context.Context, userID string) (*vo.ProfileVO, error) {
	const operation = "UserProfileService.GetProfileByUserID"

	// 1. 调用仓库层获取资料
	//    只读操作，假设 s.repo.GetProfileByUserID 签名未变。
	profileEntity, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Info("尝试获取不存在的用户资料", // 使用Info级别，因为这可能是正常业务流
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
			// 返回明确的业务错误，告知上层资料未找到。
			return nil, errors.New("用户资料不存在")
		}
		// 其他数据库查询错误
		s.logger.Error("调用仓库获取用户资料失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功获取用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	// 2. 转换并返回VO
	return profileEntityToVO(profileEntity), nil
}

// UpdateProfile 实现接口方法 (已修改更新逻辑)
func (s *userProfileService) UpdateProfile(ctx context.Context, userID string, dto *dto.UpdateProfileDTO) (*vo.ProfileVO, error) {
	const operation = "UserProfileService.UpdateProfile"

	// 1. 查询目标用户资料是否存在
	profileEntity, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试更新不存在的用户资料",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
			return nil, errors.New("要更新的用户资料不存在")
		}
		s.logger.Error("更新用户资料前查询失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// 2. 根据 DTO 中非 nil 的字段更新实体 (Patch Update Logic)
	updated := false // 标记是否有字段被实际更新

	if dto.Nickname != nil && profileEntity.Nickname != *dto.Nickname {
		// 检查 Nickname 指针是否非 nil，并且值与当前实体中的值不同
		profileEntity.Nickname = *dto.Nickname // 解引用指针获取值并更新
		updated = true
	}
	if dto.AvatarURL != nil && profileEntity.AvatarURL != *dto.AvatarURL {
		// 检查 AvatarURL 指针是否非 nil，并且值与当前实体中的值不同
		// TODO: 如果需要验证头像的 URL 格式，可以在这里添加验证逻辑
		profileEntity.AvatarURL = *dto.AvatarURL
		updated = true
	}
	if dto.Gender != nil {
		// 检查 Gender 指针是否非 nil
		// 可选：在此处验证解引用后的值是否有效 (0, 1, 2)
		genderValue := *dto.Gender
		if genderValue != enums.Unknown && genderValue != enums.Male && genderValue != enums.Female {
			s.logger.Warn("无效的性别值", zap.Any("gender", genderValue), zap.String("userID", userID))
			return nil, errors.New("无效的性别值") // 或者忽略无效值？取决于业务需求
		}
		if profileEntity.Gender != genderValue {
			profileEntity.Gender = genderValue // 解引用指针获取值并更新
			updated = true
		}
	}
	if dto.Province != nil && profileEntity.Province != *dto.Province {
		// 检查 Province 指针是否非 nil，并且值与当前实体中的值不同
		profileEntity.Province = *dto.Province
		updated = true
	}
	if dto.City != nil && profileEntity.City != *dto.City {
		// 检查 City 指针是否非 nil，并且值与当前实体中的值不同
		profileEntity.City = *dto.City
		updated = true
	}

	// 如果没有任何字段需要更新，可以直接返回当前实体对应的 VO
	if !updated {
		s.logger.Info("用户资料无需更新，未提供有效修改或值与现有数据相同",
			zap.String("operation", operation),
			zap.String("userID", userID),
		)
		// 注意：即使没有更新，返回的 VO 中的 UpdatedAt 也是从数据库读出来的旧时间
		return profileEntityToVO(profileEntity), nil
	}

	// 3. 调用仓库层更新资料
	// 仓库层的 UpdateProfile 方法通常接收整个实体。
	// 如果仓库层使用 Save，会更新所有字段（包括未改动的）。
	// 如果仓库层使用 Updates，只会更新 GORM 认为“有变化”的字段（基于原始查询结果和当前实体值的比较）。
	// 无论是 Save 还是 Updates，由于我们已经在服务层精确修改了 profileEntity，结果应该是正确的。
	if err := s.repo.UpdateProfile(ctx, profileEntity); err != nil {
		s.logger.Error("调用仓库更新用户资料失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// 4. 重新从数据库获取更新后的记录 (可选但推荐，确保返回最新数据，特别是 UpdatedAt)
	// 因为仓库层的 UpdateProfile 可能只更新部分字段，或者我们想确保返回的时间戳是数据库实际写入的。
	updatedProfileEntity, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		// 如果在这里获取失败，是一个比较严重的问题，可能表示更新后记录丢失或查询出错
		s.logger.Error("更新用户资料后重新获取记录失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		// 即使更新成功了，但无法返回最新数据，也应该报告错误
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功更新用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)

	// 5. 转换并返回更新后的 VO
	return profileEntityToVO(updatedProfileEntity), nil
}

// DeleteProfile 实现接口方法，删除用户资料。
func (s *userProfileService) DeleteProfile(ctx context.Context, userID string) error {
	const operation = "UserProfileService.DeleteProfile"

	// 1. 调用仓库层删除资料
	if err := s.repo.DeleteProfile(ctx, s.db, userID); err != nil {
		// 对于删除操作，如果记录本身未找到，通常不视为错误（幂等性）。
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试删除不存在的用户资料，操作视为成功（幂等）",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
			return nil // 返回 nil 表示操作成功或已达到期望状态。
		}
		// 其他数据库错误
		s.logger.Error("调用仓库删除用户资料失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return commonerrors.ErrSystemError
	}

	s.logger.Info("成功删除用户资料",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	return nil
}
