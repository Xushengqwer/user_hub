package oAuth

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
	"github.com/Xushengqwer/user_hub/repository/redis" // 虽然此服务目前未使用，但保持依赖注入的完整性

	"gorm.io/gorm"
)

// WechatMiniProgramService 定义了微信小程序认证相关的服务接口。
type WechatMiniProgramService interface {
	// LoginOrRegister 处理微信小程序用户的登录或自动注册流程。
	// - ctx: 请求上下文。
	// - data: 包含微信小程序前端获取的临时登录凭证 code。
	// - platform: 发起请求的客户端平台类型。
	// - 返回: 包含用户 ID 的 Userinfo、包含访问和刷新令牌的 TokenPair，以及可能发生的错误 (对上层友好)。
	LoginOrRegister(ctx context.Context, data dto.WechatMiniProgramLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// wechatMiniProgramService 是 WechatMiniProgramService 接口的实现。
type wechatMiniProgramService struct {
	identityRepo   mysql.IdentityRepository       // 身份仓库
	userRepo       mysql.UserRepository           // 用户仓库
	profileRepo    mysql.ProfileRepository        // 用户资料仓库
	tokenBlackRepo redis.TokenBlackRepo           // 令牌黑名单仓库
	jwtUtil        dependencies.JWTTokenInterface // JWT 工具
	wechatClient   dependencies.WechatClient      // 微信 API 客户端
	db             *gorm.DB                       // 数据库连接 (用于启动事务和非事务操作)
	logger         *core.ZapLogger                // 日志记录器
}

func NewWechatMiniProgramService(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	profileRepo mysql.ProfileRepository,
	tokenBlackRepo redis.TokenBlackRepo,
	jwtUtil dependencies.JWTTokenInterface,
	wechatClient dependencies.WechatClient,
	db *gorm.DB,
	logger *core.ZapLogger, // 添加 logger 参数
) WechatMiniProgramService {
	return &wechatMiniProgramService{
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		profileRepo:    profileRepo,
		tokenBlackRepo: tokenBlackRepo,
		jwtUtil:        jwtUtil,
		wechatClient:   wechatClient,
		db:             db,
		logger:         logger,
	}
}

// LoginOrRegister 实现接口方法，处理微信登录或注册。
func (s *wechatMiniProgramService) LoginOrRegister(ctx context.Context, data dto.WechatMiniProgramLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	const operation = "WechatMiniProgramService.LoginOrRegister"
	emptyUserInfo := vo.Userinfo{}
	emptyTokenPair := vo.TokenPair{}

	// 1. 调用微信 API 获取 OpenID 和 SessionKey
	openid, _, err := s.wechatClient.GetSession(ctx, data.Code)
	if err != nil {
		s.logger.Error("调用微信 GetSession 失败",
			zap.String("operation", operation),
			zap.String("code", data.Code), // 注意：code 是一次性的，记录它可能对调试有帮助，但要注意敏感性
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("微信登录凭证校验失败，请稍后重试")
	}

	// 2. 尝试根据 OpenID 查找用户身份凭证
	var userID string
	identityCredential, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.WechatMiniProgram, openid)

	if err != nil {
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 3. 用户身份不存在，执行自动注册流程
			newUserID := uuid.New().String()
			s.logger.Info("微信用户首次登录，开始自动注册",
				zap.String("operation", operation),
				zap.String("openid", openid),
				zap.String("newUserID", newUserID),
			)

			newUser := &entities.User{
				UserID:   newUserID,
				UserRole: enums.RoleUser,
				Status:   enums.StatusActive,
			}
			newIdentity := &entities.UserIdentity{
				UserID:       newUserID,
				IdentityType: myenums.WechatMiniProgram,
				Identifier:   openid,
				Credential:   "", // 微信登录通常无密码凭证，或存储 session_key (需谨慎，当前为空)
			}
			// 准备初始用户资料实体
			initialProfile := &entities.UserProfile{
				UserID: newUserID,
				// todo : Nickname 后续可以直接采取微信用户的昵称
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
				s.logger.Error("微信注册事务失败",
					zap.String("operation", operation),
					zap.String("newUserID", newUserID),
					zap.String("openid", openid),
					zap.Error(txErr),
				)
				return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy // 使用公共错误
			}
			userID = newUserID
			s.logger.Info("微信用户自动注册成功（包括用户、身份和初始资料创建）",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
		} else {
			s.logger.Error("查找微信身份信息失败",
				zap.String("operation", operation),
				zap.String("openid", openid),
				zap.Error(err),
			)
			return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
		}
	} else {
		userID = identityCredential.UserID
		s.logger.Info("微信用户已存在，直接登录",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("openid", openid),
		)
	}

	// 4. 根据 UserID 获取完整的用户信息
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
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
	}

	// 5. 检查用户状态
	if user.Status != enums.StatusActive {
		s.logger.Warn("用户尝试登录但状态异常",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Any("status", user.Status),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("用户状态异常，无法登录")
	}

	// 6. 生成令牌
	accessToken, err := s.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		s.logger.Error("生成访问令牌失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
	}
	refreshToken, err := s.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		s.logger.Error("生成刷新令牌失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
	}

	// 7. 成功返回
	s.logger.Info("微信登录/注册成功",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.Any("platform", platform),
	)
	userInfo := vo.Userinfo{UserID: user.UserID}
	tokenPair := vo.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return userInfo, tokenPair, nil
}
