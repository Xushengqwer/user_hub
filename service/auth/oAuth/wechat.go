package oAuth

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"user_hub/common/dependencies"
	"user_hub/models/dto"
	"user_hub/models/entities"
	"user_hub/models/enums"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/repository/redis"
	"user_hub/userError"

	"gorm.io/gorm"
)

// WechatMiniProgram 定义微信小程序认证服务接口
type WechatMiniProgram interface {
	// LoginOrRegister 使用微信小程序授权码进行登录或注册
	// - 输入: ctx 用于上下文控制，data 包含微信授权码，platform 表示客户端平台
	// - 输出: Userinfo 返回用户信息，TokenPair 返回访问令牌和刷新令牌，error 表示可能的错误
	LoginOrRegister(ctx context.Context, data dto.WechatMiniProgramLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// wechatMiniProgram 实现 WechatMiniProgram 接口的结构体
type wechatMiniProgram struct {
	identityRepo   mysql.IdentityRepository         // 用于查询和创建用户身份
	userRepo       mysql.UserRepository             // 用于创建和查询用户信息
	tokenBlackRepo redis.TokenBlackRepo             // 用于管理令牌黑名单
	jwtUtil        dependencies.JWTUtilityInterface // 用于生成和解析 JWT 令牌
	wechatClient   dependencies.WechatClient        // 用于调用微信 API
	db             *gorm.DB                         // 用于事务管理
}

// NewWechatMiniProgram 创建 WechatMiniProgram 实例，通过依赖注入初始化
// - 输入: 各仓库和工具的实例
// - 输出: WechatMiniProgram 接口实例
func NewWechatMiniProgram(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	tokenBlackRepo redis.TokenBlackRepo,
	jwtUtil dependencies.JWTUtilityInterface,
	wechatClient dependencies.WechatClient,
	db *gorm.DB,
) WechatMiniProgram {
	return &wechatMiniProgram{
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		tokenBlackRepo: tokenBlackRepo,
		jwtUtil:        jwtUtil,
		wechatClient:   wechatClient,
		db:             db,
	}
}

// LoginOrRegister 使用微信小程序授权码进行登录或注册
func (m *wechatMiniProgram) LoginOrRegister(ctx context.Context, data dto.WechatMiniProgramLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	// 1. 使用微信授权码调用微信 API，获取 openid 和 session_key
	// - 调用 WechatClient 的 GetSession 方法获取微信用户标识
	// - 如果微信 API 调用失败，返回服务器内部错误
	openid, _, err := m.wechatClient.GetSession(ctx, data.Code)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 2. 根据 openid 查询用户身份
	// - 使用 IdentityRepository 检查该微信用户是否已注册
	// - 如果找到，直接获取 UserID
	var userID string
	identity, err := m.identityRepo.GetIdentityByTypeAndIdentifier(ctx, enums.WechatMiniProgram, openid)
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			// 3. 用户不存在，自动注册新用户
			// - 生成唯一的 UserID，使用 UUID
			userID = uuid.New().String()

			// - 创建 User 记录，设置默认角色和状态
			user := &entities.User{
				UserID:   userID,
				UserRole: enums.User,   // 默认普通用户角色
				Status:   enums.Active, // 默认活跃状态
			}

			// - 创建 UserIdentity 记录，存储微信 openid
			identity := &entities.UserIdentity{
				UserID:       userID,
				IdentityType: enums.WechatMiniProgram, // 身份类型为微信小程序
				Identifier:   openid,                  // 微信用户唯一标识
				Credential:   "",                      // 微信登录无需密码，留空
			}

			// - 使用事务确保 User 和 UserIdentity 同时创建成功
			err = m.db.Transaction(func(tx *gorm.DB) error {
				if err := m.userRepo.CreateUser(ctx, user); err != nil {
					return err // 创建失败，事务回滚
				}
				if err := m.identityRepo.CreateIdentity(ctx, identity); err != nil {
					return err // 创建失败，事务回滚
				}
				return nil // 创建成功，事务提交
			})
			if err != nil {
				return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
			}
		} else {
			// - 查询过程中发生其他错误，返回服务器内部错误
			return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
		}
	}

	// 4. 获取用户信息（已注册或刚注册的用户）
	// - 如果用户已存在，identity 中包含 UserID
	// - 如果刚注册，userID 已生成并赋值
	if identity == nil {
		// - 如果 identity 为 nil，使用刚生成的 userID
		// - 注意：逻辑上不会走到此分支，因为注册后 identity 已赋值
	} else {
		userID = identity.UserID
	}

	// 5. 查询完整的用户信息
	// - 根据 UserID 获取 User 记录，用于生成令牌
	user, err := m.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 6. 生成访问令牌和刷新令牌
	// - 使用 JWTUtility 生成 token，平台由控制器动态传入
	// - 注意：微信小程序通常固定为 Wechat 平台，但这里保持灵活性
	accessToken, err := m.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}
	refreshToken, err := m.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 7. 返回用户信息和令牌
	// - Userinfo 返回 UserID，TokenPair 包含访问令牌和刷新令牌
	return vo.Userinfo{UserID: user.UserID}, vo.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
