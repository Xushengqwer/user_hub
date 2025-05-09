package identity

import (
	"context"
	"errors"
	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"go.uber.org/zap"                       // 引入 zap 用于日志字段

	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	"github.com/Xushengqwer/user_hub/models/enums"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/utils" // 引入密码工具

	"gorm.io/gorm"
)

// UserIdentityService 定义了管理用户多种身份标识（如账号密码、微信、手机号等）的服务接口。
// 设计目的:
// - 将用户身份凭证的管理与核心用户属性、具体的认证流程（如登录、注册）解耦。
// - 提供一个统一的入口来操作不同类型的用户身份数据，便于未来扩展新的身份验证方式。
// 使用场景:
// - 管理员或用户自身管理其关联的多种登录方式。
// - 例如：用户绑定新的邮箱作为登录方式、修改某个登录方式的密码、解绑某个社交登录。
type UserIdentityService interface {
	// CreateIdentity 为指定用户创建一个新的身份标识记录。
	// 使用场景:
	//  - 用户在注册后，希望添加另一种登录方式（例如，已用手机注册，想再绑定邮箱密码登录）。
	//  - 管理员为用户手动添加某种登录凭证。
	// 参数:
	//  - ctx: 请求上下文，用于控制超时和取消。
	//  - dto: 包含创建新身份所需的数据，如用户ID、身份类型、唯一标识符和凭证。
	// 返回:
	//  - *vo.IdentityVO: 成功创建的身份信息的视图对象。
	//  - error: 操作过程中发生的任何错误，可能是业务错误或系统错误。
	CreateIdentity(ctx context.Context, dto *dto.CreateIdentityDTO) (*vo.IdentityVO, error)

	// UpdateIdentity 更新指定身份ID的凭证信息。
	// 使用场景:
	//  - 用户修改其账号密码登录方式的密码。
	//  - 系统更新了某个OAuth身份的访问令牌（虽然此场景下通常是更新凭证，但具体取决于OAuth流程）。
	// 参数:
	//  - identityID: 要更新的身份记录的数据库主键ID。
	//  - dto: 包含新凭证信息的数据传输对象。
	// 返回:
	//  - *vo.IdentityVO: 更新后的身份信息的视图对象。
	//  - error: 操作过程中发生的任何错误。
	UpdateIdentity(ctx context.Context, identityID uint, dto *dto.UpdateIdentityDTO) (*vo.IdentityVO, error)

	// DeleteIdentity 删除指定ID的用户身份记录。
	// 使用场景:
	//  - 用户解绑某个登录方式（例如，不再使用微信登录）。
	//  - 管理员移除某个用户的特定登录凭证。
	// 参数:
	//  - identityID: 要删除的身份记录的数据库主键ID。
	// 返回:
	//  - error: 操作过程中发生的任何错误。
	DeleteIdentity(ctx context.Context, identityID uint) error

	// GetIdentitiesByUserID 检索指定用户ID关联的所有身份记录。
	// 使用场景:
	//  - 用户在个人资料页面查看自己已绑定的所有登录方式。
	//  - 管理员后台查看某个用户的全部身份凭证信息（不含敏感凭证内容）。
	// 参数:
	//  - userID: 要查询的用户ID。
	// 返回:
	//  - []*vo.IdentityVO: 用户身份信息视图对象的列表。如果用户没有任何身份记录，返回空列表。
	//  - error: 操作过程中发生的任何错误。
	GetIdentitiesByUserID(ctx context.Context, userID string) ([]*vo.IdentityVO, error)

	// GetIdentityTypesByUserID 检索指定用户ID所拥有的所有身份类型。
	// 使用场景:
	//  - 系统内部判断用户是否拥有特定类型的登录方式，例如，决定是否显示“通过手机号找回密码”的选项。
	//  - 快速展示用户已绑定的登录方式类型概览。
	// 参数:
	//  - userID: 要查询的用户ID。
	// 返回:
	//  - []enums.IdentityType: 用户身份类型的枚举列表。如果用户没有任何身份记录，返回空列表。
	//  - error: 操作过程中发生的任何错误。
	GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error)
}

// userIdentityService 是 UserIdentityService 接口的实现。
// 它封装了与用户身份相关的业务逻辑和数据持久化操作。
type userIdentityService struct {
	repo mysql.IdentityRepository // repo: 身份数据仓库，负责与数据库直接交互。
	db   *gorm.DB                 // db: GORM数据库连接实例。主要用于将原始连接传递给仓库层方法，
	// 因为此服务中的每个方法通常代表一个独立的、原子性的操作单元。
	// 如果这些方法需要被编排进一个更大的、跨多个服务方法或仓库方法的事务，
	// 那么事务的开启和管理应在更高层（如应用服务编排层或特定的业务流程服务）进行，
	// 并将事务性 `*gorm.DB` (即 `tx`) 传递给底层的仓库方法。
	logger *core.ZapLogger // logger: 日志记录器，用于记录操作信息和错误。
}

// NewUserIdentityService 创建一个新的 userIdentityService 实例。
// 设计原因:
// - 采用依赖注入方式，将仓库、数据库连接和日志记录器作为参数传入。
// - 这种设计提高了代码的可测试性（可以mock依赖）和灵活性（方便替换实现）。
func NewUserIdentityService(
	repo mysql.IdentityRepository,
	db *gorm.DB,
	logger *core.ZapLogger,
) UserIdentityService {
	return &userIdentityService{
		repo:   repo,
		db:     db,
		logger: logger,
	}
}

// entityToVO 是一个内部辅助函数，用于将数据库实体 `entities.UserIdentity` 转换为对外暴露的视图对象 `vo.IdentityVO`。
// 设计原因:
// - 实现领域模型与视图模型的隔离，避免直接暴露数据库实体结构给上层或外部。
// - 可以在转换过程中进行数据裁剪（如不暴露敏感的 `Credential` 字段）或格式化。
func entityToVO(identity *entities.UserIdentity) *vo.IdentityVO {
	if identity == nil {
		return nil
	}
	return &vo.IdentityVO{
		IdentityID:   identity.IdentityID,
		UserID:       identity.UserID,
		IdentityType: identity.IdentityType,
		Identifier:   identity.Identifier,
		CreatedAt:    identity.CreatedAt,
		UpdatedAt:    identity.UpdatedAt,
		// 注意：vo.IdentityVO 通常不包含 Credential (凭证) 字段，以保证安全。
	}
}

// CreateIdentity 实现接口方法，为用户创建新的身份标识。
func (s *userIdentityService) CreateIdentity(ctx context.Context, dto *dto.CreateIdentityDTO) (*vo.IdentityVO, error) {
	const operation = "UserIdentityService.CreateIdentity" // 用于日志和错误追踪的操作标识

	// 1. 准备身份实体 (Data Preparation and Validation)
	//    - 对于账号密码类型的身份，凭证（密码）在存储前必须进行哈希处理。
	//    - 其他类型的身份凭证可能不需要特殊处理，或有其自身的验证逻辑（例如OAuth token）。
	credential := dto.Credential
	if dto.IdentityType == enums.AccountPassword { // 假设 enums.AccountPassword 已在公共模块定义
		hashedPassword, err := utils.SetPassword(dto.Credential) // 使用密码工具进行哈希
		if err != nil {
			s.logger.Error("创建身份时密码加密失败",
				zap.String("operation", operation),
				zap.String("userID", dto.UserID),
				zap.Any("identityType", dto.IdentityType),
				zap.Error(err), // 记录原始加密错误
			)
			// 密码加密失败是系统内部问题，向上层返回通用系统错误。
			return nil, commonerrors.ErrSystemError
		}
		credential = hashedPassword
	}

	identityEntity := &entities.UserIdentity{
		UserID:       dto.UserID,
		IdentityType: dto.IdentityType,
		Identifier:   dto.Identifier,
		Credential:   credential, // 使用处理后（可能已加密）的凭证
	}

	// 2. 调用仓库层创建身份记录
	//    - 传递 s.db (原始数据库连接)，因为此服务方法本身被视为一个原子操作。
	//    - 假设 s.repo.CreateIdentity 签名已更新为 `CreateIdentity(ctx, db, entity)`。
	if err := s.repo.CreateIdentity(ctx, s.db, identityEntity); err != nil {
		s.logger.Error("调用仓库创建身份失败",
			zap.String("operation", operation),
			zap.String("userID", dto.UserID),
			zap.Any("identityType", dto.IdentityType),
			zap.String("identifier", dto.Identifier),
			zap.Error(err), // 记录来自仓库的原始错误
		)
		// 此处可以根据 err 的具体类型判断是否为唯一约束冲突等特定数据库错误，
		// 并返回更具体的业务错误，例如：
		// if errors.Is(err, some_specific_db_duplicate_error) {
		//     return nil, errors.New("该身份标识（如邮箱或手机号）已被其他用户使用")
		// }
		// 当前简化为返回通用系统错误。
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功创建用户身份",
		zap.String("operation", operation),
		zap.Uint("identityID", identityEntity.IdentityID), // 记录新生成的身份ID
		zap.String("userID", identityEntity.UserID),
	)

	// 3. 将创建成功的实体转换为视图对象并返回
	return entityToVO(identityEntity), nil
}

// UpdateIdentity 实现接口方法，更新指定身份的凭证。
func (s *userIdentityService) UpdateIdentity(ctx context.Context, identityID uint, dto *dto.UpdateIdentityDTO) (*vo.IdentityVO, error) {
	const operation = "UserIdentityService.UpdateIdentity"

	// 1. 查询目标身份记录是否存在
	//    - 这是必要的步骤，以确保我们正在更新一个实际存在的记录，并获取其当前状态（如身份类型）。
	//    - 假设 s.repo.GetIdentityByID 是只读操作，使用 s.repo 内部的原始 s.db。
	//      如果它也需要 db 参数，则应传入 s.db。
	identityEntity, err := s.repo.GetIdentityByID(ctx, identityID)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试更新不存在的身份记录",
				zap.String("operation", operation),
				zap.Uint("identityID", identityID),
			)
			// 返回明确的业务错误，告知上层记录未找到。
			return nil, errors.New("要更新的身份记录不存在")
		}
		// 其他数据库查询错误
		s.logger.Error("更新身份前查询记录失败",
			zap.String("operation", operation),
			zap.Uint("identityID", identityID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// 2. 准备新的凭证
	//    - 同样，如果身份类型是账号密码，新凭证需要加密。
	newCredential := dto.Credential
	if identityEntity.IdentityType == enums.AccountPassword {
		hashedPassword, err := utils.SetPassword(dto.Credential)
		if err != nil {
			s.logger.Error("更新身份时密码加密失败",
				zap.String("operation", operation),
				zap.Uint("identityID", identityID),
				zap.Error(err),
			)
			return nil, commonerrors.ErrSystemError
		}
		newCredential = hashedPassword
	}
	identityEntity.Credential = newCredential // 更新实体中的凭证

	// 3. 调用仓库层更新身份记录
	if err := s.repo.UpdateIdentity(ctx, identityEntity); err != nil {
		s.logger.Error("调用仓库更新身份失败",
			zap.String("operation", operation),
			zap.Uint("identityID", identityID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功更新用户身份凭证",
		zap.String("operation", operation),
		zap.Uint("identityID", identityID),
	)

	// 4. 将更新后的实体转换为视图对象并返回
	return entityToVO(identityEntity), nil
}

// DeleteIdentity 实现接口方法，删除指定的用户身份。
func (s *userIdentityService) DeleteIdentity(ctx context.Context, identityID uint) error {
	const operation = "UserIdentityService.DeleteIdentity"

	// 1. 调用仓库层删除身份记录
	if err := s.repo.DeleteIdentity(ctx, s.db, identityID); err != nil {
		// 对于删除操作，如果记录本身未找到 (ErrRepoNotFound)，通常不视为一个需要向上层报错的“失败”。
		// 操作是幂等的：删除一个不存在的东西和成功删除它，最终状态是一样的（它不存在）。
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试删除不存在的身份记录，操作视为成功（幂等）",
				zap.String("operation", operation),
				zap.Uint("identityID", identityID),
			)
			return nil // 返回 nil 表示操作成功或已达到期望状态。
		}
		// 其他类型的数据库错误则需要记录并向上层报告。
		s.logger.Error("调用仓库删除身份失败",
			zap.String("operation", operation),
			zap.Uint("identityID", identityID),
			zap.Error(err),
		)
		return commonerrors.ErrSystemError
	}

	s.logger.Info("成功删除用户身份",
		zap.String("operation", operation),
		zap.Uint("identityID", identityID),
	)
	return nil
}

// GetIdentitiesByUserID 实现接口方法，获取用户的所有身份信息。
func (s *userIdentityService) GetIdentitiesByUserID(ctx context.Context, userID string) ([]*vo.IdentityVO, error) {
	const operation = "UserIdentityService.GetIdentitiesByUserID"

	// 1. 调用仓库层获取身份实体列表
	//    - 这是一个只读操作，通常可以直接使用仓库内部的 s.db，除非有特殊需要在事务中读取。
	//    - 假设 s.repo.GetIdentitiesByUserID 签名未变。
	identityEntities, err := s.repo.GetIdentitiesByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("调用仓库获取用户身份列表失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	// 2. 将实体列表转换为视图对象列表
	//    - 如果没有记录，会返回一个空的 vo.IdentityVO 切片，这是期望的行为。
	identityVOs := make([]*vo.IdentityVO, 0, len(identityEntities))
	for _, entity := range identityEntities {
		identityVOs = append(identityVOs, entityToVO(entity))
	}

	s.logger.Info("成功获取用户身份列表",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.Int("count", len(identityVOs)), // 记录获取到的数量
	)
	return identityVOs, nil
}

// GetIdentityTypesByUserID 实现接口方法，获取用户的所有身份类型。
func (s *userIdentityService) GetIdentityTypesByUserID(ctx context.Context, userID string) ([]enums.IdentityType, error) {
	const operation = "UserIdentityService.GetIdentityTypesByUserID"

	// 1. 调用仓库层获取身份类型列表
	//    - 只读操作。
	//    - 假设 s.repo.GetIdentityTypesByUserID 签名未变。
	identityTypes, err := s.repo.GetIdentityTypesByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("调用仓库获取用户身份类型列表失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return nil, commonerrors.ErrSystemError
	}

	s.logger.Info("成功获取用户身份类型列表",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.Int("count", len(identityTypes)), // 记录获取到的数量
	)
	return identityTypes, nil
}
