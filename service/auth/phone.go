package auth

import (
	"context"
	"errors"
	redisv8 "github.com/go-redis/redis/v8" // 标准 Redis 客户端库，别名为 redisv8
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

// Phone 定义手机号认证服务接口
type Phone interface {
	// LoginOrRegister 使用手机号和验证码进行登录或注册
	// - ctx: 用于上下文控制
	// - data: 包含手机号和验证码
	// - platform: 表示客户端平台类型
	// - 返回: 用户信息、令牌对及可能的错误
	LoginOrRegister(ctx context.Context, data dto.PhoneLoginOrRegisterData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// phone 实现 Phone 接口的结构体
type phone struct {
	identityRepo mysql.IdentityRepository         // 用户身份仓库
	userRepo     mysql.UserRepository             // 用户信息仓库
	codeRepo     redis.CodeRepo                   // 验证码存储仓库（自定义）
	jwtUtil      dependencies.JWTUtilityInterface // JWT 令牌工具
	db           *gorm.DB                         // 数据库连接，用于事务
}

// NewPhone 创建 Phone 实例
// - 通过依赖注入初始化各组件
func NewPhone(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	codeRepo redis.CodeRepo,
	jwtUtil dependencies.JWTUtilityInterface,
	db *gorm.DB,
) Phone {
	return &phone{
		identityRepo: identityRepo,
		userRepo:     userRepo,
		codeRepo:     codeRepo,
		jwtUtil:      jwtUtil,
		db:           db,
	}
}

// LoginOrRegister 使用手机号和验证码进行登录或注册
func (p *phone) LoginOrRegister(ctx context.Context, data dto.PhoneLoginOrRegisterData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	// 1. 验证验证码
	// - 从 Redis 获取存储的验证码，使用自定义 codeRepo
	storedCode, err := p.codeRepo.GetCaptcha(ctx, data.Phone)
	if err != nil {
		// - 使用 redisv8.Nil 检查键不存在的情况（验证码过期或未设置）
		if errors.Is(err, redisv8.Nil) {
			return vo.Userinfo{}, vo.TokenPair{}, userError.ErrCaptchaExpired // 验证码过期或不存在
		}
		// - 其他 Redis 查询错误，返回服务器内部错误
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}
	if storedCode != data.Code {
		// - 用户提交的验证码与存储的不匹配
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrCaptchaInvalid
	}
	// - 验证码验证通过，删除验证码防止重用
	if err := p.codeRepo.DeleteCaptcha(ctx, data.Phone); err != nil {
		// - 删除失败不影响主流程，建议记录日志
		// TODO: 记录日志
	}

	// 2. 检查用户是否已注册
	// - 根据手机号查询用户身份
	var userID string
	identity, err := p.identityRepo.GetIdentityByTypeAndIdentifier(ctx, enums.Phone, data.Phone)
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			// 3. 用户不存在，自动注册
			// - 生成唯一的 UserID
			userID = uuid.New().String()

			// - 创建 User 记录，设置默认角色和状态
			user := &entities.User{
				UserID:   userID,
				UserRole: enums.User,   // 默认角色：普通用户
				Status:   enums.Active, // 默认状态：活跃
			}

			// - 创建 UserIdentity 记录，存储手机号
			identity := &entities.UserIdentity{
				UserID:       userID,
				IdentityType: enums.Phone, // 身份类型：手机号
				Identifier:   data.Phone,  // 手机号
				Credential:   "",          // 无密码，留空
			}

			// - 使用事务确保用户和身份记录一致性
			err = p.db.Transaction(func(tx *gorm.DB) error {
				if err := p.userRepo.CreateUser(ctx, user); err != nil {
					return err // 创建失败，事务回滚
				}
				if err := p.identityRepo.CreateIdentity(ctx, identity); err != nil {
					return err // 创建失败，事务回滚
				}
				return nil // 创建成功，事务提交
			})
			if err != nil {
				return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
			}
		} else {
			// - 查询身份失败，返回服务器内部错误
			return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
		}
	} else {
		// - 用户已存在，使用现有 UserID
		userID = identity.UserID
	}

	// 4. 获取用户信息
	// - 根据 UserID 查询完整用户信息
	user, err := p.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 5. 生成令牌
	// - 使用 JWT 工具生成访问令牌和刷新令牌
	accessToken, err := p.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}
	refreshToken, err := p.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 6. 返回结果
	// - 返回用户信息和令牌对
	return vo.Userinfo{UserID: user.UserID}, vo.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
