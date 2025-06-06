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
	myenums "github.com/Xushengqwer/user_hub/models/enums" // 确保 myenums 别名被正确使用
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/repository/redis"
	// "github.com/Xushengqwer/user_hub/service/profile" // 不再需要 profileService

	"gorm.io/gorm"
)

// PhoneAuthService 定义了基于手机号和验证码认证的服务接口。
type PhoneAuthService interface {
	// LoginOrRegister 处理用户使用手机号和验证码进行登录或自动注册的逻辑。
	// - ctx: 请求上下文。
	// - data: 包含手机号和验证码的 DTO。
	// - platform: 发起请求的客户端平台类型。
	// - 返回: 包含用户 ID 的 Userinfo、包含访问和刷新令牌的 TokenPair，以及可能发生的业务错误或系统错误。
	LoginOrRegister(ctx context.Context, data dto.PhoneLoginOrRegisterData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// phoneAuthService 是 PhoneAuthService 接口的实现。
type phoneAuthService struct {
	identityRepo mysql.IdentityRepository       // 身份仓库
	userRepo     mysql.UserRepository           // 用户仓库
	profileRepo  mysql.ProfileRepository        // 用户资料仓库
	codeRepo     redis.CodeRepo                 // 验证码仓库
	jwtUtil      dependencies.JWTTokenInterface // JWT 工具
	db           *gorm.DB                       // 数据库连接
	logger       *core.ZapLogger                // 日志记录器
}

func NewPhoneAuthService(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	profileRepo mysql.ProfileRepository,
	codeRepo redis.CodeRepo,
	jwtUtil dependencies.JWTTokenInterface,
	db *gorm.DB,
	logger *core.ZapLogger,
) PhoneAuthService {
	return &phoneAuthService{
		identityRepo: identityRepo,
		userRepo:     userRepo,
		profileRepo:  profileRepo,
		codeRepo:     codeRepo,
		jwtUtil:      jwtUtil,
		db:           db,
		logger:       logger,
	}
}

// LoginOrRegister 实现接口方法，处理手机号登录或注册。
func (s *phoneAuthService) LoginOrRegister(ctx context.Context, data dto.PhoneLoginOrRegisterData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	const operation = "PhoneAuthService.LoginOrRegister"
	emptyUserInfo := vo.Userinfo{}
	emptyTokenPair := vo.TokenPair{}

	// 1. 验证验证码
	storedCode, err := s.codeRepo.GetCaptcha(ctx, data.Phone)
	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("验证码错误或已过期",
				zap.String("operation", operation),
				zap.String("phone", data.Phone),
			)
			return emptyUserInfo, emptyTokenPair, errors.New("验证码错误或已过期")
		}
		s.logger.Error("获取验证码失败",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	if storedCode != data.Code {
		s.logger.Warn("用户提交的验证码不匹配",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
		)
		return emptyUserInfo, emptyTokenPair, errors.New("验证码错误或已过期")
	}

	if err := s.codeRepo.DeleteCaptcha(ctx, data.Phone); err != nil {
		s.logger.Error("删除已使用的验证码失败",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
			zap.Error(err),
		)
	}
	s.logger.Info("验证码校验通过", zap.String("operation", operation), zap.String("phone", data.Phone))

	// 2. 检查用户是否已通过该手机号注册
	var userID string
	identityCredential, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.Phone, data.Phone)

	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 3. 用户身份不存在，执行自动注册流程
			newUserID := uuid.New().String()
			s.logger.Info("手机号用户首次登录，开始自动注册",
				zap.String("operation", operation),
				zap.String("phone", data.Phone),
				zap.String("newUserID", newUserID),
			)

			newUser := &entities.User{
				UserID:   newUserID,
				UserRole: enums.RoleUser,
				Status:   enums.StatusActive,
			}
			newIdentity := &entities.UserIdentity{
				UserID:       newUserID,
				IdentityType: myenums.Phone,
				Identifier:   data.Phone,
				Credential:   "", // 手机号登录通常无密码
			}
			// 准备初始用户资料实体
			initialProfile := &entities.UserProfile{
				UserID:   newUserID,
				Nickname: data.Phone,
			}

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
				s.logger.Error("手机号注册事务失败",
					zap.String("operation", operation),
					zap.String("newUserID", newUserID),
					zap.String("phone", data.Phone),
					zap.Error(txErr),
				)
				return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
			}
			userID = newUserID
			s.logger.Info("手机号用户自动注册成功（包括用户、身份和初始资料创建）",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
		} else {
			s.logger.Error("查找手机号身份信息失败",
				zap.String("operation", operation),
				zap.String("phone", data.Phone),
				zap.Error(err),
			)
			return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
		}
	} else {
		userID = identityCredential.UserID
		s.logger.Info("手机号用户已存在，直接登录",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("phone", data.Phone),
		)
	}

	// 4. 根据 UserID 获取完整的用户信息 )
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户信息失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Error("数据不一致：身份存在但核心用户记录未找到",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
			return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
		}
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 5. 检查用户状态
	if user.Status != enums.StatusActive {
		s.logger.Warn("尝试登录但用户状态异常",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Any("status", user.Status),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("用户状态异常，无法登录")
	}

	// 6. 生成令牌
	accessToken, err := s.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		s.logger.Error("生成访问令牌失败",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}
	refreshToken, err := s.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		s.logger.Error("生成刷新令牌失败",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 7. 成功完成登录或注册
	s.logger.Info("手机号登录/注册成功",
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
