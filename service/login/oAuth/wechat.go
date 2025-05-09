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

	dependencies "github.com/Xushengqwer/user_hub/dependencies" // 重命名导入以避免与包名冲突
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/entities"
	myenums "github.com/Xushengqwer/user_hub/models/enums"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/repository/redis"

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
	tokenBlackRepo redis.TokenBlackRepo           // 令牌黑名单仓库
	jwtUtil        dependencies.JWTTokenInterface // JWT 工具
	wechatClient   dependencies.WechatClient      // 微信 API 客户端
	db             *gorm.DB                       // 数据库连接 (用于启动事务和非事务操作)
	logger         *core.ZapLogger                // 日志记录器
}

// NewWechatMiniProgramService 创建一个新的 wechatMiniProgramService 实例。
// - 通过依赖注入初始化所有必需的组件，包括日志记录器。
func NewWechatMiniProgramService(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	tokenBlackRepo redis.TokenBlackRepo,
	jwtUtil dependencies.JWTTokenInterface,
	wechatClient dependencies.WechatClient,
	db *gorm.DB,
	logger *core.ZapLogger, // 添加 logger 参数
) WechatMiniProgramService { // 返回接口类型
	return &wechatMiniProgramService{ // 返回结构体指针
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		tokenBlackRepo: tokenBlackRepo,
		jwtUtil:        jwtUtil,
		wechatClient:   wechatClient,
		db:             db,
		logger:         logger, // 存储 logger
	}
}

// LoginOrRegister 实现接口方法，处理微信登录或注册。
func (s *wechatMiniProgramService) LoginOrRegister(ctx context.Context, data dto.WechatMiniProgramLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	const operation = "WechatLoginOrRegister" // 操作名称，用于日志和错误信息
	emptyUserInfo := vo.Userinfo{}
	emptyTokenPair := vo.TokenPair{}

	// 1. 调用微信 API 获取 OpenID 和 SessionKey
	openid, _, err := s.wechatClient.GetSession(ctx, data.Code)
	if err != nil {
		s.logger.Error("调用微信 GetSession 失败",
			zap.String("operation", operation),
			zap.String("code", data.Code),
			zap.Error(err),
		)
		return emptyUserInfo, emptyTokenPair, fmt.Errorf("微信登录凭证校验失败，请稍后重试")
	}

	// 2. 尝试根据 OpenID 查找用户身份凭证
	var userID string
	// 注意：GetIdentityByTypeAndIdentifier 通常是只读操作，不需要在事务内执行，
	// 也不需要传递事务 tx。它使用 s.identityRepo 内部的 s.db (原始连接)。
	identityCredential, err := s.identityRepo.GetIdentityByTypeAndIdentifier(ctx, myenums.WechatMiniProgram, openid)

	if err != nil {
		// 检查是否是“未找到”错误
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
				Credential:   "",
			}

			// *** 关键修正：在事务块内调用仓库方法时，传递 tx ***
			txErr := s.db.Transaction(func(tx *gorm.DB) error {
				// 调用 CreateUser 时传入 tx
				if err := s.userRepo.CreateUser(ctx, tx, newUser); err != nil {
					return fmt.Errorf("事务中创建用户失败: %w", err)
				}
				// 调用 CreateIdentity 时传入 tx
				if err := s.identityRepo.CreateIdentity(ctx, tx, newIdentity); err != nil {
					return fmt.Errorf("事务中创建身份失败: %w", err)
				}
				return nil // 事务成功
			})

			// 检查事务是否出错
			if txErr != nil {
				s.logger.Error("微信注册事务失败",
					zap.String("operation", operation),
					zap.String("newUserID", newUserID),
					zap.String("openid", openid),
					zap.Error(txErr),
				)
				return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
			}
			userID = newUserID
			s.logger.Info("微信用户自动注册成功",
				zap.String("operation", operation),
				zap.String("userID", userID),
			)
		} else {
			// 查找身份时发生其他数据库错误
			s.logger.Error("查找微信身份信息失败",
				zap.String("operation", operation),
				zap.String("openid", openid),
				zap.Error(err),
			)
			return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
		}
	} else {
		// 4. 用户身份已存在
		userID = identityCredential.UserID
		s.logger.Info("微信用户已存在，直接登录",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("openid", openid),
		)
	}

	// 5. 根据 UserID 获取完整的用户信息
	//    GetUserByID 通常也是只读操作，可以使用原始 s.db
	//    如果你的 GetUserByID 实现也修改为接收 *gorm.DB，这里就需要传入 s.db
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
		return emptyUserInfo, emptyTokenPair, commonerrors.ErrServiceBusy
	}

	// 检查用户状态
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
