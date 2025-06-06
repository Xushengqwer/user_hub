package auth

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装

	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/google/uuid"
	"go.uber.org/zap" // 引入 zap 用于日志字段

	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	myenums "github.com/Xushengqwer/user_hub/models/enums"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/repository/redis"
	"github.com/Xushengqwer/user_hub/utils" // 引入密码工具

	"gorm.io/gorm"
)

// AccountService 定义了基于账号密码认证的服务接口。
type AccountService interface {
	// Register 处理用户使用账号密码进行注册的逻辑。
	// - ctx: 请求上下文。
	// - data: 包含账号、密码和确认密码的注册信息 DTO。
	// - 返回: 包含新用户 ID 的 Userinfo，以及可能发生的业务错误或系统错误。
	// - 注意: 注册成功后不自动登录，不返回令牌。
	Register(ctx context.Context, data dto.AccountRegisterData) (vo.Userinfo, error)

	// Login 处理用户使用账号密码进行登录的逻辑。
	// - ctx: 请求上下文。
	// - data: 包含账号和密码的登录信息 DTO。
	// - platform: 发起请求的客户端平台类型。
	// - 返回: 包含用户 ID 的 Userinfo、包含访问和刷新令牌的 TokenPair，以及可能发生的业务错误或系统错误。
	Login(ctx context.Context, data dto.AccountLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// accountService 是 AccountService 接口的实现。
type accountService struct {
	identityRepo   mysql.IdentityRepository // 身份仓库
	userRepo       mysql.UserRepository     // 用户仓库
	tokenBlackRepo redis.TokenBlackRepo     // 令牌黑名单仓库 (Login 中未使用，但保持注入)
	profileRepo    mysql.ProfileRepository
	jwtUtil        dependencies.JWTTokenInterface // JWT 工具
	db             *gorm.DB                       // 数据库连接
	logger         *core.ZapLogger                // 日志记录器
}

func NewAccountService(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	profileRepo mysql.ProfileRepository,
	tokenBlackRepo redis.TokenBlackRepo,
	jwtUtil dependencies.JWTTokenInterface,
	db *gorm.DB,
	logger *core.ZapLogger, // 注入 logger
) AccountService { // 返回接口类型
	return &accountService{ // 返回结构体指针
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		profileRepo:    profileRepo,
		tokenBlackRepo: tokenBlackRepo,
		jwtUtil:        jwtUtil,
		db:             db,
		logger:         logger, // 存储 logger
	}
}

// Register 实现接口方法，处理用户注册。
func (s *accountService) Register(ctx context.Context, data dto.AccountRegisterData) (vo.Userinfo, error) {
	const operation = "AccountService.Register" // 修改操作名称以反映服务层
	emptyUserInfo := vo.Userinfo{}

	// 1. 基本校验：密码与确认密码是否一致
	if data.Password != data.ConfirmPassword {
		s.logger.Warn("注册时密码与确认密码不一致", zap.String("operation", operation), zap.String("account", data.Account))
		return emptyUserInfo, errors.New("密码和确认密码不一致，请检查输入")
	}

	// 2. 检查账号是否已存在
	_, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.AccountPassword, data.Account)
	if err == nil {
		s.logger.Warn("尝试注册已存在的账号",
			zap.String("operation", operation),
			zap.String("account", data.Account),
		)
		return emptyUserInfo, errors.New("账号已存在，请直接登录")
	} else if !errors.Is(err, commonerrors.ErrRepoNotFound) {
		s.logger.Error("检查账号是否存在时查询失败",
			zap.String("operation", operation),
			zap.String("account", data.Account),
			zap.Error(err),
		)
		return emptyUserInfo, commonerrors.ErrSystemError
	}

	// 3. 准备注册信息
	userID := uuid.New().String()
	s.logger.Info("账号不存在，开始新用户注册流程",
		zap.String("operation", operation),
		zap.String("account", data.Account),
		zap.String("newUserID", userID),
	)

	hashedPassword, err := utils.SetPassword(data.Password)
	if err != nil {
		s.logger.Error("密码加密失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyUserInfo, commonerrors.ErrSystemError
	}

	newUser := &entities.User{
		UserID:   userID,
		UserRole: enums.RoleUser,     // 默认为普通用户
		Status:   enums.StatusActive, // 默认状态为活跃
	}
	newIdentity := &entities.UserIdentity{
		UserID:       userID,
		IdentityType: myenums.AccountPassword,
		Identifier:   data.Account,
		Credential:   hashedPassword,
	}
	// 准备初始用户资料实体，只包含 UserID
	initialProfile := &entities.UserProfile{
		UserID:   userID,
		Nickname: data.Account,
		// 其他字段（如 AvatarURL, Gender, Province, City）将使用数据库默认值或保持为空
	}

	// 4. 使用事务创建用户、身份和初始资料
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.userRepo.CreateUser(ctx, tx, newUser); err != nil {
			return fmt.Errorf("事务中创建用户失败: %w", err)
		}
		if err := s.identityRepo.CreateIdentity(ctx, tx, newIdentity); err != nil {
			return fmt.Errorf("事务中创建身份失败: %w", err)
		}
		// 在事务中创建初始用户资料
		if err := s.profileRepo.CreateProfile(ctx, tx, initialProfile); err != nil {
			return fmt.Errorf("事务中创建初始用户资料失败: %w", err)
		}
		return nil // 事务成功
	})

	if txErr != nil {
		s.logger.Error("账号注册事务失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("account", data.Account),
			zap.Error(txErr),
		)
		return emptyUserInfo, commonerrors.ErrSystemError
	}

	// 5. 注册成功
	s.logger.Info("账号注册成功（包括用户、身份和初始资料创建）",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.String("account", data.Account),
	)
	return vo.Userinfo{UserID: userID}, nil
}

// Login 实现接口方法，处理用户登录。
func (s *accountService) Login(ctx context.Context, data dto.AccountLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	const operation = "AccountLogin"
	emptyUserInfo := vo.Userinfo{}
	emptyTokenPair := vo.TokenPair{}

	// 1. 根据账号查找身份凭证
	identityCredential, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.AccountPassword, data.Account)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("尝试登录不存在的账号",
				zap.String("operation", operation),
				zap.String("account", data.Account),
			)
			return emptyUserInfo, emptyTokenPair, errors.New("账号不存在或密码错误")
		}
		s.logger.Error("登录时查找账号身份失败",
			zap.String("operation", operation),
			zap.String("account", data.Account),
			zap.Error(err),
		)
		// 查询失败返回系统错误
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 2. 校验密码
	if err := utils.CheckPassword(identityCredential.Credential, data.Password); err != nil {
		s.logger.Warn("登录密码错误",
			zap.String("operation", operation),
			zap.String("userID", identityCredential.UserID),
			zap.String("account", data.Account),
		)
		return emptyUserInfo, emptyTokenPair, errors.New("账号不存在或密码错误")
	}

	// 3. 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, identityCredential.UserID)
	if err != nil {
		s.logger.Error("登录时获取用户信息失败",
			zap.String("operation", operation),
			zap.String("userID", identityCredential.UserID),
			zap.Error(err),
		)
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			return emptyUserInfo, emptyTokenPair, fmt.Errorf("用户数据异常，请联系管理员")
		}
		// 获取用户信息失败返回系统错误
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 4. 检查用户状态
	if user.Status != enums.StatusActive {
		s.logger.Warn("尝试登录但用户状态异常",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Any("status", user.Status),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("用户状态异常，无法登录")
	}

	// 5. 生成令牌
	accessToken, err := s.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		s.logger.Error("生成访问令牌失败",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Error(err),
		)
		// 生成令牌失败返回系统错误
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}
	refreshToken, err := s.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		s.logger.Error("生成刷新令牌失败",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Error(err),
		)
		// 生成令牌失败返回系统错误
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 6. 登录成功
	s.logger.Info("账号登录成功",
		zap.String("operation", operation),
		zap.String("userID", user.UserID),
		zap.Any("platform", platform),
	)
	userInfo := vo.Userinfo{UserID: user.UserID}
	tokenPair := vo.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return userInfo, tokenPair, nil
}
