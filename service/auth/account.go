package auth

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
	"user_hub/utils"

	"gorm.io/gorm"
)

// Account 定义账号密码认证服务接口
type Account interface {
	// Register 使用账号密码注册
	// - 输入: ctx 用于上下文控制，data 包含注册信息
	// - 输出: Userinfo 返回用户信息，error 表示可能的错误
	// - 注意: 注册后不返回令牌，用户需单独登录
	Register(ctx context.Context, data dto.AccountRegisterData) (vo.Userinfo, error)

	// Login 使用账号密码登录
	// - 输入: ctx 用于上下文控制，data 包含登录信息，platform 表示客户端平台
	// - 输出: Userinfo 返回用户信息，TokenPair 返回访问令牌和刷新令牌，error 表示可能的错误
	Login(ctx context.Context, data dto.AccountLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error)
}

// account 实现 Account 接口的结构体
type account struct {
	identityRepo   mysql.IdentityRepository         // 用于查询和创建用户身份
	userRepo       mysql.UserRepository             // 用于创建和查询用户信息
	tokenBlackRepo redis.TokenBlackRepo             // 用于管理令牌黑名单
	jwtUtil        dependencies.JWTUtilityInterface // 用于生成和解析 JWT 令牌
	db             *gorm.DB                         // 用于事务管理
}

// NewAccount 创建 Account 实例，通过依赖注入初始化
// - 输入: 各仓库和工具的实例
// - 输出: Account 接口实例
func NewAccount(
	identityRepo mysql.IdentityRepository,
	userRepo mysql.UserRepository,
	tokenBlackRepo redis.TokenBlackRepo,
	jwtUtil dependencies.JWTUtilityInterface,
	db *gorm.DB,
) Account {
	return &account{
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		tokenBlackRepo: tokenBlackRepo,
		jwtUtil:        jwtUtil,
		db:             db,
	}
}

// Register 使用账号密码注册
func (a *account) Register(ctx context.Context, data dto.AccountRegisterData) (vo.Userinfo, error) {
	// 1. 检查密码和确认密码是否一致
	// - 虽然控制器层可能已校验，但服务层再确认一次，确保逻辑完整
	if data.Password != data.ConfirmPassword {
		return vo.Userinfo{}, errors.New("密码和确认密码不一致，请检查输入")
	}

	// 2. 查询账号是否已存在
	// - 使用 IdentityRepository 检查，避免重复注册
	_, err := a.identityRepo.GetIdentityByTypeAndIdentifier(ctx, enums.AccountPassword, data.Account)
	if err == nil {
		// - 如果查询无错，说明账号已存在，返回错误
		return vo.Userinfo{}, errors.New("账号已存在，请直接登录")
	} else if !errors.Is(err, userError.ErrIdentityNotFound) {
		// - 如果不是“未找到身份”的错误，说明查询过程异常，返回服务器错误
		return vo.Userinfo{}, userError.ErrServerInternal
	}

	// 3. 账号不存在，开始注册流程
	// - 生成唯一的用户 ID，使用 UUID
	userID := uuid.New().String()

	// - 创建用户记录，设置默认角色和状态
	user := &entities.User{
		UserID:   userID,
		UserRole: enums.User,   // 默认注册为普通用户角色
		Status:   enums.Active, // 默认状态为活跃
	}

	// - 创建用户身份记录，存储账号和加密密码
	hashedPassword, err := utils.SetPassword(data.Password) // 使用 bcrypt 加密密码
	if err != nil {
		// - 加密失败，返回服务器错误
		return vo.Userinfo{}, userError.ErrServerInternal
	}
	identity := &entities.UserIdentity{
		UserID:       userID,
		IdentityType: enums.AccountPassword, // 身份类型为账号密码登录
		Identifier:   data.Account,          // 账号（如用户名或邮箱）
		Credential:   hashedPassword,        // 加密后的密码
	}

	// - 使用事务，确保 User 和 UserIdentity 同时创建成功
	err = a.db.Transaction(func(tx *gorm.DB) error {
		if err := a.userRepo.CreateUser(ctx, user); err != nil {
			return err // 创建失败，事务回滚
		}
		if err := a.identityRepo.CreateIdentity(ctx, identity); err != nil {
			return err // 创建失败，事务回滚
		}
		return nil // 创建成功，事务提交
	})
	if err != nil {
		// - 事务执行失败，返回服务器错误
		return vo.Userinfo{}, userError.ErrServerInternal
	}

	// 4. 注册成功，返回用户信息
	// - 只返回 Userinfo，不生成令牌，用户需单独登录
	return vo.Userinfo{UserID: userID}, nil
}

// Login 使用账号密码登录
func (a *account) Login(ctx context.Context, data dto.AccountLoginData, platform enums.Platform) (vo.Userinfo, vo.TokenPair, error) {
	// 1. 查询账号是否存在
	// - 使用 IdentityRepository 检查账号是否已注册
	identity, err := a.identityRepo.GetIdentityByTypeAndIdentifier(ctx, enums.AccountPassword, data.Account)
	if err != nil {
		if errors.Is(err, userError.ErrIdentityNotFound) {
			// - 账号不存在，返回未注册错误
			return vo.Userinfo{}, vo.TokenPair{}, errors.New("账号不存在，请先注册")
		}
		// - 查询过程中发生其他错误，返回服务器错误
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 2. 验证密码是否正确
	// - 使用 bcrypt 校验用户输入的密码与存储的哈希密码
	if err := utils.CheckPassword(identity.Credential, data.Password); err != nil {
		// - 密码不匹配，返回错误提示
		return vo.Userinfo{}, vo.TokenPair{}, errors.New("密码错误，请检查输入")
	}

	// 3. 获取用户信息
	// - 根据 UserID 查询完整用户信息，用于生成令牌
	user, err := a.userRepo.GetUserByID(ctx, identity.UserID)
	if err != nil {
		// - 查询失败，返回服务器错误
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 4. 生成访问令牌和刷新令牌
	// - 使用 JWTUtility 生成 token，平台由控制器动态传入
	accessToken, err := a.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}
	refreshToken, err := a.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		return vo.Userinfo{}, vo.TokenPair{}, userError.ErrServerInternal
	}

	// 5. 登录成功，返回用户信息和令牌
	// - Userinfo 返回 UserID，TokenPair 包含访问令牌和刷新令牌
	return vo.Userinfo{UserID: user.UserID}, vo.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
