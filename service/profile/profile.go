package profile

import (
	"context"
	"errors"
	"fmt"
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/enums"
	"io"

	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"go.uber.org/zap"                       // 引入 zap 用于日志字段

	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"

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
	// UpdateProfile 更新自己的个人资料信息。
	// 使用场景:
	//  - 用户（包括普通用户和管理员）在个人资料设置页面修改自己的昵称、头像、性别、地区等。
	// 参数:
	//  - userID: 要更新资料的用户ID。
	//  - dto: 包含待更新资料字段的 DTO。DTO中的字段为可选更新，服务会根据DTO中提供的非空/非零值进行更新。
	// 返回:
	//  - *vo.ProfileVO: 更新后的用户资料的视图对象。
	//  - error: 操作过程中发生的任何错误。
	UpdateProfile(ctx context.Context, userID string, dto *dto.UpdateProfileDTO) (*vo.ProfileVO, error)

	// UploadAndSetAvatar 上传用户头像到COS，并更新用户资料中的头像URL。
	// 参数:
	//  - userID: 要更新头像的用户ID。
	//  - fileName: 上传文件的原始名称，用于提取扩展名。
	//  - fileReader: 包含文件内容的 io.Reader。
	//  - fileSize: 文件大小（字节）。
	// 返回:
	//  - string: 成功上传后头像的公开访问URL。
	//  - error: 操作过程中发生的任何错误。
	UploadAndSetAvatar(ctx context.Context, userID string, fileName string, fileReader io.Reader, fileSize int64) (string, error)

	// GetMyAccountDetail 获取当前认证用户的聚合账户详情（核心信息 + 资料）。
	// 参数:
	//  - ctx: 请求上下文。
	//  - userID: 当前认证用户的ID。
	// 返回:
	//  - *vo.MyAccountDetailVO: 包含用户核心信息和资料的视图对象。
	//  - error: 操作过程中发生的任何错误。
	GetMyAccountDetail(ctx context.Context, userID string) (*vo.MyAccountDetailVO, error)
}

// userProfileService 是 UserProfileService 接口的实现。
type userProfileService struct {
	userRepo  mysql.UserRepository            // 用户核心信息仓库
	repo      mysql.ProfileRepository         // repo: 用户资料数据仓库。
	db        *gorm.DB                        // db: GORM数据库连接实例，用于传递给仓库层的写操作方法。
	logger    *core.ZapLogger                 // logger: 日志记录器。
	cosClient dependencies.COSClientInterface // <--- 新增此字段
}

func NewUserProfileService(
	userRepo mysql.UserRepository,
	repo mysql.ProfileRepository,
	db *gorm.DB,
	logger *core.ZapLogger,
	cosClient dependencies.COSClientInterface, // <--- 新增此参数
) UserProfileService {
	return &userProfileService{
		userRepo:  userRepo,
		repo:      repo,
		db:        db,
		logger:    logger,
		cosClient: cosClient,
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

// UpdateProfile 实现接口方法
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

// UploadAndSetAvatar 方法修改：直接更新实体并保存
func (s *userProfileService) UploadAndSetAvatar(ctx context.Context, userID string, fileName string, fileReader io.Reader, fileSize int64) (string, error) {
	const operation = "UserProfileService.UploadAndSetAvatar"
	s.logger.Info("开始上传并设置用户头像", zap.String("operation", operation), zap.String("userID", userID), zap.String("fileName", fileName), zap.Int64("fileSize", fileSize))

	// 1. 上传头像到 COS
	avatarURL, err := s.cosClient.UploadUserAvatar(ctx, userID, fileName, fileReader, fileSize)
	if err != nil {
		s.logger.Error("上传头像到腾讯云 COS 失败", zap.String("operation", operation), zap.String("userID", userID), zap.String("fileName", fileName), zap.Error(err))
		return "", fmt.Errorf("上传头像到腾讯云 COS 服务失败: %w", commonerrors.ErrThirdPartyServiceError)
	}
	s.logger.Info("头像成功上传到 COS", zap.String("operation", operation), zap.String("userID", userID), zap.String("avatarURL", avatarURL))

	// 2. 获取当前用户资料实体
	profileEntity, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		// 如果用户资料不存在，这可能是一个错误，因为理论上用户注册时应已创建。
		// 或者，如果允许在没有 profile 的情况下上传头像（然后创建profile），逻辑会不同。
		// 当前假设 profile 必须存在。
		s.logger.Error("更新头像URL前获取用户资料失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 根据之前的讨论，如果网关确保用户存在，那么这里 profile 不存在应视为内部错误
			return "", fmt.Errorf("用户资料不存在，无法更新头像: %w", commonerrors.ErrSystemError)
		}
		return "", commonerrors.ErrSystemError
	}

	// 3.直接修改实体中的 AvatarURL
	if profileEntity.AvatarURL == avatarURL {
		s.logger.Info("新的头像URL与现有URL相同，无需更新数据库", zap.String("operation", operation), zap.String("userID", userID), zap.String("avatarURL", avatarURL))
		return avatarURL, nil // 如果URL未变，则无需更新数据库
	}
	profileEntity.AvatarURL = avatarURL

	// 4. 调用仓库层更新（保存）整个实体
	// 注意：s.repo.UpdateProfile 接收的是整个实体，它的内部实现是 GORM 的 Save，它会更新所有字段。
	// 如果是 Updates，它会更新有变化的字段。
	// 通常，对于部分更新，先获取实体，修改字段，然后 Save 是常见做法。
	if err := s.repo.UpdateProfile(ctx, profileEntity); err != nil {
		s.logger.Error("更新用户资料中的头像URL失败（仓库层）", zap.String("operation", operation), zap.String("userID", userID), zap.String("newAvatarURL", avatarURL), zap.Error(err))
		// 错误处理策略：
		// 此时图片已上传到 COS，但数据库更新失败。
		// - 选项1: 尝试删除已上传的 COS 对象（会增加代码复杂性，需要 s.cosClient.DeleteObject）。
		// - 选项2: 返回错误，让用户重试。下次上传可能会覆盖或创建新对象，取决于 UploadUserAvatar 的对象键生成逻辑。
		// - 选项3: 记录严重错误，可能需要人工介入。
		// 当前选择选项2，简单返回错误。
		return "", commonerrors.ErrSystemError
	}

	s.logger.Info("成功更新用户资料中的头像URL", zap.String("operation", operation), zap.String("userID", userID), zap.String("newAvatarURL", avatarURL))
	return avatarURL, nil
}

// GetMyAccountDetail 实现接口方法，获取当前用户的聚合账户详情。
func (s *userProfileService) GetMyAccountDetail(ctx context.Context, userID string) (*vo.MyAccountDetailVO, error) {
	const operation = "UserProfileService.GetMyAccountDetail"
	s.logger.Info("开始获取用户账户详情", zap.String("operation", operation), zap.String("userID", userID))

	var userEntity *entities.User
	var profileEntity *entities.UserProfile
	var err error

	// 1. 获取核心用户信息
	userEntity, err = s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取核心用户信息失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 按照之前的讨论，如果网关确保用户存在，这里找不到核心用户记录应视为严重的数据不一致
			return nil, fmt.Errorf("核心用户记录不存在，数据异常: %w", commonerrors.ErrSystemError)
		}
		return nil, commonerrors.ErrSystemError
	}

	// 2. 获取用户资料信息
	profileEntity, err = s.repo.GetProfileByUserID(ctx, userID) // s.repo 是 profileRepo
	if err != nil {
		s.logger.Error("获取用户资料信息失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 用户注册时应已创建初始 profile。如果这里找不到，也视为数据问题。
			s.logger.Warn("用户资料记录未找到，可能数据初始化不完整或被异常删除", zap.String("operation", operation), zap.String("userID", userID))
			return nil, fmt.Errorf("用户资料记录不存在，数据异常: %w", commonerrors.ErrSystemError)
		}
		return nil, commonerrors.ErrSystemError
	}

	// 3. 组装 MyAccountDetailVO
	accountDetail := &vo.MyAccountDetailVO{
		UserID:    userEntity.UserID,
		UserRole:  userEntity.UserRole, // 使用 commonEnums.UserRole
		Status:    userEntity.Status,   // 使用 commonEnums.UserStatus
		Nickname:  profileEntity.Nickname,
		AvatarURL: profileEntity.AvatarURL,
		Gender:    profileEntity.Gender, // 使用 projectEnums.Gender
		Province:  profileEntity.Province,
		City:      profileEntity.City,
		CreatedAt: userEntity.CreatedAt,    // 通常使用核心用户的创建时间
		UpdatedAt: profileEntity.UpdatedAt, // 可以使用 profile 的更新时间，或两者中较新的一个
	}

	s.logger.Info("成功获取用户账户详情", zap.String("operation", operation), zap.String("userID", userID))
	return accountDetail, nil
}
