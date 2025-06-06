package userManage

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装

	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"

	"gorm.io/gorm"
)

// UserManageService 定义了管理核心用户账户（User 实体）的服务接口。
// 设计目的:
// - 封装用户账户的生命周期管理，包括创建、状态变更（如拉黑）、信息更新和删除。
// - 协调与用户身份凭证（Identity）和详细资料（Profile）相关的操作，确保数据一致性。
type UserManageService interface {
	// CreateUser 创建一个新的核心用户记录。通常由其他服务（如注册服务）调用。
	// 参数:
	//  - ctx: 请求上下文。
	//  - dto: 包含新用户角色和状态的 DTO。用户 ID 由服务内部生成。
	// 返回:
	//  - *vo.UserVO: 成功创建的用户信息的视图对象。
	//  - error: 操作过程中发生的任何错误。
	CreateUser(ctx context.Context, dto *dto.CreateUserDTO) (*vo.UserVO, error)

	// GetUserByID 根据用户 ID 检索核心用户信息。
	// 参数:
	//  - userID: 要查询的用户 ID。
	// 返回:
	//  - *vo.UserVO: 用户信息的视图对象。如果用户不存在，返回业务错误。
	//  - error: 操作过程中发生的任何错误。
	GetUserByID(ctx context.Context, userID string) (*vo.UserVO, error)

	// GetUserProfileByAdmin (管理员权限) 根据用户 ID 检索指定用户的详细资料信息。
	// 参数:
	//  - ctx: 请求上下文。
	//  - userID: 要查询的用户 ID。
	// 返回:
	//  - *vo.ProfileVO: 用户资料的视图对象。如果用户资料不存在，返回业务错误。
	//  - error: 操作过程中发生的任何错误。
	GetUserProfileByAdmin(ctx context.Context, userID string) (*vo.ProfileVO, error)

	// UpdateUser 更新指定用户的核心信息（目前主要是角色和状态）。
	// 参数:
	//  - userID: 要更新的用户 ID。
	//  - dto: 包含待更新字段的 DTO。服务会根据 DTO 中提供的非零值进行更新。
	// 返回:
	//  - *vo.UserVO: 更新后的用户信息的视图对象。
	//  - error: 操作过程中发生的任何错误。
	UpdateUser(ctx context.Context, userID string, dto *dto.UpdateUserDTO) (*vo.UserVO, error)

	// DeleteUser （软）删除指定用户及其所有关联的身份和资料信息。
	// 此操作将在一个数据库事务中执行，以确保原子性。
	// 参数:
	//  - userID: 要删除的用户 ID。
	// 返回:
	//  - error: 操作过程中发生的任何错误。
	DeleteUser(ctx context.Context, userID string) error

	// BlackUser 将指定用户标记为“拉黑”状态。
	// 参数:
	//  - userID: 要拉黑的用户 ID。
	// 返回:
	//  - error: 操作过程中发生的任何错误。
	BlackUser(ctx context.Context, userID string) error
}

// userService 是 UserManageService 接口的实现。
type userService struct {
	userRepo     mysql.UserRepository     // userRepo: 用户数据仓库。
	identityRepo mysql.IdentityRepository // identityRepo: 用户身份数据仓库。
	profileRepo  mysql.ProfileRepository  // profileRepo: 用户资料数据仓库。
	db           *gorm.DB                 // db: GORM数据库连接实例，用于启动事务和传递给仓库方法。
	logger       *core.ZapLogger          // logger: 日志记录器。
}

// NewUserService 创建一个新的 userService 实例。
// 设计原因:
// - 依赖注入确保了服务的可测试性和灵活性。
func NewUserService(
	userRepo mysql.UserRepository,
	identityRepo mysql.IdentityRepository, // 注入 identityRepo
	profileRepo mysql.ProfileRepository, // 注入 profileRepo
	db *gorm.DB,
	logger *core.ZapLogger,
) UserManageService {
	return &userService{
		userRepo:     userRepo,
		identityRepo: identityRepo, // 存储 identityRepo
		profileRepo:  profileRepo,  // 存储 profileRepo
		db:           db,
		logger:       logger,
	}
}

// userEntityToVO 是一个内部辅助函数，用于将数据库实体 `entities.User` 转换为对外暴露的视图对象 `vo.UserVO`。
func userEntityToVO(user *entities.User) *vo.UserVO {
	if user == nil {
		return nil
	}
	return &vo.UserVO{
		UserID:    user.UserID,
		UserRole:  user.UserRole,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// CreateUser 实现接口方法，创建新用户。
func (s *userService) CreateUser(ctx context.Context, dto *dto.CreateUserDTO) (*vo.UserVO, error) {
	const operation = "UserManageService.CreateUser"
	userID := uuid.New().String()
	s.logger.Info("开始创建新用户",
		zap.String("operation", operation),
		zap.String("newUserID", userID),
		zap.Any("role", dto.UserRole),
		zap.Any("status", dto.Status),
	)

	userEntity := &entities.User{
		UserID:   userID,
		UserRole: dto.UserRole,
		Status:   dto.Status,
	}

	if err := s.userRepo.CreateUser(ctx, s.db, userEntity); err != nil {
		s.logger.Error("调用仓库创建用户失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}
	s.logger.Info("成功创建用户", zap.String("operation", operation), zap.String("userID", userID))

	// *** 新增：创建成功后，重新从数据库获取记录以获取正确时间戳 ***
	createdUserEntity, err := s.userRepo.GetUserByID(ctx, userID) // 使用新用户的 userID
	if err != nil {
		// 如果刚创建就获取失败，记录错误但可能仍返回原始实体（不含正确时间戳）或错误
		s.logger.Error("创建用户后获取记录失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		// 这里可以选择是返回之前的 userEntity (时间戳为零值)，还是直接返回错误
		// 为了接口一致性，这里选择返回错误，表明无法确认最终状态
		return nil, commonerrors.ErrSystemError // 或者返回 userEntityToVO(userEntity) 但告知调用方时间戳可能不准
	}

	s.logger.Info("成功创建用户", zap.String("operation", operation), zap.String("userID", userID))
	// *** 修改：使用从数据库读回的实体进行转换 ***

	return userEntityToVO(createdUserEntity), nil
}

// GetUserByID 实现接口方法，获取用户信息。
func (s *userService) GetUserByID(ctx context.Context, userID string) (*vo.UserVO, error) {
	const operation = "UserManageService.GetUserByID"
	userEntity, err := s.userRepo.GetUserByID(ctx, userID) // 假设 GetUserByID 不需要 db 参数
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Info("尝试获取不存在的用户", zap.String("operation", operation), zap.String("userID", userID))
			return nil, errors.New("用户不存在")
		}
		s.logger.Error("调用仓库获取用户失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		return nil, commonerrors.ErrSystemError
	}
	s.logger.Info("成功获取用户信息", zap.String("operation", operation), zap.String("userID", userID))
	return userEntityToVO(userEntity), nil
}

// GetUserProfileByAdmin (管理员权限) 根据用户 ID 检索指定用户的详细资料信息。
func (s *userService) GetUserProfileByAdmin(ctx context.Context, userID string) (*vo.ProfileVO, error) {
	const operation = "UserManageService.GetUserProfileByAdmin"
	s.logger.Info("管理员开始获取用户资料", zap.String("operation", operation), zap.String("targetUserID", userID))

	// 1. 调用 profileRepo 获取用户资料实体
	//    s.profileRepo 是在 NewUserService 中注入的 ProfileRepository 实例
	profileEntity, err := s.profileRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Info("管理员尝试获取不存在的用户资料",
				zap.String("operation", operation),
				zap.String("targetUserID", userID),
			)
			return nil, errors.New("指定用户的资料不存在") // 返回业务错误
		}
		// 其他数据库查询错误
		s.logger.Error("管理员调用仓库获取用户资料失败",
			zap.String("operation", operation),
			zap.String("targetUserID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError // 返回系统错误
	}

	s.logger.Info("管理员成功获取用户资料",
		zap.String("operation", operation),
		zap.String("targetUserID", userID),
	)

	// 2. 将实体转换为视图对象并返回
	//    使用我们在此文件中定义的 userProfileEntityToVO 辅助函数
	return userProfileEntityToVO(profileEntity), nil
}

// UpdateUser 实现接口方法，更新用户信息。
func (s *userService) UpdateUser(ctx context.Context, userID string, dto *dto.UpdateUserDTO) (*vo.UserVO, error) {
	const operation = "UserManageService.UpdateUser"
	userEntity, err := s.userRepo.GetUserByID(ctx, userID) // 先获取
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试更新不存在的用户", zap.String("operation", operation), zap.String("userID", userID))
			return nil, errors.New("要更新的用户不存在")
		}
		s.logger.Error("更新用户前查询失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		return nil, commonerrors.ErrSystemError
	}

	updated := false
	if dto.UserRole != 0 && userEntity.UserRole != dto.UserRole {
		userEntity.UserRole = dto.UserRole
		updated = true
	}
	if dto.Status != 0 && userEntity.Status != dto.Status {
		userEntity.Status = dto.Status
		updated = true
	}

	if !updated {
		s.logger.Info("用户信息无需更新", zap.String("operation", operation), zap.String("userID", userID))
		return userEntityToVO(userEntity), nil
	}

	if err := s.userRepo.UpdateUser(ctx, userEntity); err != nil {
		s.logger.Error("调用仓库更新用户失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		return nil, commonerrors.ErrSystemError
	}

	// *** 新增：更新成功后，重新从数据库获取最新记录 ***
	updatedUserEntity, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("更新用户后重新获取记录失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		return nil, commonerrors.ErrSystemError // 报告错误
	}

	s.logger.Info("成功更新用户信息", zap.String("operation", operation), zap.String("userID", userID))
	return userEntityToVO(updatedUserEntity), nil
}

// DeleteUser 实现接口方法，事务性地软删除用户及其关联的身份和资料。
func (s *userService) DeleteUser(ctx context.Context, userID string) error {
	const operation = "UserManageService.DeleteUserCascade" // 操作名可以更具体
	s.logger.Info("开始删除用户及其所有关联数据（事务性）",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)

	// 开启数据库事务
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 软删除核心用户记录
		//    仓库方法 DeleteUser 接收事务对象 tx
		if repoErr := s.userRepo.DeleteUser(ctx, tx, userID); repoErr != nil {
			// 如果用户本就不存在，对于删除操作通常视为成功（幂等）
			if errors.Is(repoErr, commonerrors.ErrRepoNotFound) {
				s.logger.Warn("尝试删除不存在的核心用户记录，操作视为成功（幂等）",
					zap.String("operation", operation),
					zap.String("userID", userID),
					zap.Error(repoErr), // 记录一下原始的 NotFound 错误
				)
				// 注意：如果后续操作依赖于用户必须存在才能删除，这里可能需要返回错误。
				// 但如果目标是确保用户及其数据最终不存在，NotFound 不是问题。
				// 考虑到我们还要删除关联数据，如果用户不存在，关联数据也应该不存在或无需删除。
				// 为了简化，如果核心用户删除时报 NotFound，我们继续尝试删除关联数据，或者直接认为事务成功。
				// 这里选择继续，让后续步骤处理各自的 NotFound。
			} else {
				// 其他类型的错误导致删除核心用户失败
				s.logger.Error("事务中软删除核心用户失败",
					zap.String("operation", operation),
					zap.String("userID", userID),
					zap.Error(repoErr),
				)
				return fmt.Errorf("删除核心用户记录失败: %w", repoErr) // 导致事务回滚
			}
		}

		// 2. 软删除该用户的所有身份信息
		//    调用 identityRepo 的 DeleteIdentitiesByUserID 方法，传入事务对象 tx
		//    假设该方法内部处理了记录不存在的情况（通常是 RowsAffected=0，error=nil）
		if repoErr := s.identityRepo.DeleteIdentitiesByUserID(ctx, tx, userID); repoErr != nil {
			// 此处不检查 ErrRepoNotFound，因为删除0条记录是正常情况
			s.logger.Error("事务中软删除用户身份信息失败",
				zap.String("operation", operation),
				zap.String("userID", userID),
				zap.Error(repoErr),
			)
			return fmt.Errorf("删除用户身份信息失败: %w", repoErr) // 导致事务回滚
		}
		s.logger.Info("事务中：已尝试删除用户身份信息", zap.String("operation", operation), zap.String("userID", userID))

		// 3. 软删除该用户的资料信息
		//    调用 profileRepo 的 DeleteProfile 方法，传入事务对象 tx
		//    该方法按 UserID 删除，如果 Profile 不存在，仓库层应处理 NotFound (可能返回 nil 或 ErrRepoNotFound)
		if repoErr := s.profileRepo.DeleteProfile(ctx, tx, userID); repoErr != nil {
			if errors.Is(repoErr, commonerrors.ErrRepoNotFound) {
				s.logger.Info("事务中：用户资料本就不存在，无需删除",
					zap.String("operation", operation),
					zap.String("userID", userID),
				)
				// 资料不存在不是错误，继续
			} else {
				s.logger.Error("事务中软删除用户资料信息失败",
					zap.String("operation", operation),
					zap.String("userID", userID),
					zap.Error(repoErr),
				)
				return fmt.Errorf("删除用户资料信息失败: %w", repoErr) // 导致事务回滚
			}
		}
		s.logger.Info("事务中：已尝试删除用户资料信息", zap.String("operation", operation), zap.String("userID", userID))

		// 所有操作成功，事务将自动提交
		return nil
	})

	// 检查事务执行结果
	if err != nil {
		// 事务失败的日志已在匿名函数内部或调用链中记录
		s.logger.Error("删除用户及其关联数据事务最终失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err), // 记录事务返回的顶层错误
		)
		return commonerrors.ErrSystemError // 向上层返回通用系统错误
	}

	s.logger.Info("成功删除用户及其所有关联数据（事务性）",
		zap.String("operation", operation),
		zap.String("userID", userID),
	)
	return nil
}

// BlackUser 实现接口方法，拉黑用户。
func (s *userService) BlackUser(ctx context.Context, userID string) error {
	const operation = "UserManageService.BlackUser"
	// 假设 s.repo.BlackUser 签名已更新为 BlackUser(ctx, db, userID)
	if err := s.userRepo.BlackUser(ctx, userID); err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试拉黑不存在的用户", zap.String("operation", operation), zap.String("userID", userID))
			return errors.New("要拉黑的用户不存在")
		}
		s.logger.Error("调用仓库拉黑用户失败", zap.String("operation", operation), zap.String("userID", userID), zap.Error(err))
		return commonerrors.ErrSystemError
	}
	s.logger.Info("成功拉黑用户", zap.String("operation", operation), zap.String("userID", userID))
	return nil
}

// userProfileEntityToVO 是一个内部辅助函数，用于将数据库实体 `entities.UserProfile` 转换为对外暴露的视图对象 `vo.ProfileVO`。
// 注意：此函数与之前在 profileService 中的 profileEntityToVO 功能相同。
// 如果 vo.ProfileVO 的定义没有改变，这个转换逻辑也应该保持一致。
func userProfileEntityToVO(profile *entities.UserProfile) *vo.ProfileVO {
	if profile == nil {
		return nil
	}
	return &vo.ProfileVO{
		UserID:    profile.UserID,
		Nickname:  profile.Nickname,
		AvatarURL: profile.AvatarURL,
		Gender:    profile.Gender, // 确保 entities.UserProfile 和 vo.ProfileVO 中的 Gender 类型一致或可转换
		Province:  profile.Province,
		City:      profile.City,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}
