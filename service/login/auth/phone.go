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

	"user_hub/dependencies"
	"user_hub/models/dto"
	"user_hub/models/entities"
	myenums "user_hub/models/enums"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/repository/redis"

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
	codeRepo     redis.CodeRepo                 // 验证码仓库
	jwtUtil      dependencies.JWTTokenInterface // JWT 工具
	db           *gorm.DB                       // 数据库连接
	logger       *core.ZapLogger                // 日志记录器
}

// NewPhoneAuthService 创建一个新的 phoneAuthService 实例。
// - 通过依赖注入初始化所有必需的组件。
func NewPhoneAuthService(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	codeRepo redis.CodeRepo,
	jwtUtil dependencies.JWTTokenInterface,
	db *gorm.DB,
	logger *core.ZapLogger, // 注入 logger
) PhoneAuthService { // 返回接口类型
	return &phoneAuthService{ // 返回结构体指针
		identityRepo: identityRepo,
		userRepo:     userRepo,
		codeRepo:     codeRepo,
		jwtUtil:      jwtUtil,
		db:           db,
		logger:       logger, // 存储 logger
	}
}

// LoginOrRegister 实现接口方法，处理手机号登录或注册。
func (s *phoneAuthService) LoginOrRegister(ctx context.Context, data dto.PhoneLoginOrRegisterData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	const operation = "PhoneLoginOrRegister"
	emptyUserInfo := vo.Userinfo{}
	emptyTokenPair := vo.TokenPair{}

	// 1. 验证验证码
	storedCode, err := s.codeRepo.GetCaptcha(ctx, data.Phone)
	if err != nil {
		// 检查是否是仓库层返回的“未找到”错误
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			s.logger.Warn("验证码错误或已过期",
				zap.String("operation", operation),
				zap.String("phone", data.Phone), // 注意脱敏
			)
			// 返回明确的业务错误给用户
			return emptyUserInfo, emptyTokenPair, errors.New("验证码错误或已过期")
		}
		// 其他 Redis 查询错误
		s.logger.Error("获取验证码失败",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
			zap.Error(err),
		)
		// 返回系统错误
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 校验用户提交的验证码是否匹配
	if storedCode != data.Code {
		s.logger.Warn("用户提交的验证码不匹配",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
			// 不记录用户提交的错误验证码，避免日志过多
		)
		return emptyUserInfo, emptyTokenPair, errors.New("验证码错误或已过期") // 统一提示
	}

	// 验证码使用后立即删除，防止重放攻击
	if err := s.codeRepo.DeleteCaptcha(ctx, data.Phone); err != nil {
		// 删除失败通常不影响主流程，但需要记录日志以供排查
		s.logger.Error("删除已使用的验证码失败",
			zap.String("operation", operation),
			zap.String("phone", data.Phone),
			zap.Error(err),
		)
		// 不向用户返回错误，继续执行
	}
	s.logger.Info("验证码校验通过", zap.String("operation", operation), zap.String("phone", data.Phone))

	// 2. 检查用户是否已通过该手机号注册
	var userID string
	identityCredential, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.Phone, data.Phone)

	if err != nil {
		// 检查是否是“未找到”错误
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

			// 使用数据库事务确保原子性
			txErr := s.db.Transaction(func(tx *gorm.DB) error {
				// *** 关键：调用仓库方法时传入事务 tx ***
				if err := s.userRepo.CreateUser(ctx, tx, newUser); err != nil {
					return fmt.Errorf("事务中创建用户失败: %w", err)
				}
				if err := s.identityRepo.CreateIdentity(ctx, tx, newIdentity); err != nil {
					return fmt.Errorf("事务中创建身份失败: %w", err)
				}
				return nil // 事务成功
			})

			// 检查事务结果
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
			s.logger.Info("手机号用户自动注册成功",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
		} else {
			// 查找身份时发生其他数据库错误
			s.logger.Error("查找手机号身份信息失败",
				zap.String("operation", operation),
				zap.String("phone", data.Phone),
				zap.Error(err),
			)
			return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
		}
	} else {
		// 4. 用户身份已存在
		userID = identityCredential.UserID
		s.logger.Info("手机号用户已存在，直接登录",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("phone", data.Phone),
		)
	}

	// 5. 根据 UserID 获取完整的用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("获取用户信息失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			return emptyUserInfo, emptyTokenPair, fmt.Errorf("未能找到用户信息，数据可能异常")
		}
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrSystemError
	}

	// 6. 检查用户状态
	if user.Status != enums.StatusActive {
		s.logger.Warn("尝试登录但用户状态异常",
			zap.String("operation", operation),
			zap.String("userID", user.UserID),
			zap.Any("status", user.Status),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("用户状态异常，无法登录")
	}

	// 7. 生成令牌
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

	// 8. 成功完成登录或注册
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
