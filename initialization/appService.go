package initialization

import (
	"user_hub/repository/mysql"
	"user_hub/repository/redis"
	"user_hub/service/auth"
	"user_hub/service/auth/oAuth"
	"user_hub/service/identity"
	"user_hub/service/profile"
	"user_hub/service/token"
	"user_hub/service/user"
	"user_hub/service/userList"
)

// AppServices 封装所有服务层实例 + 一个redis的code仓库存储验证码
type AppServices struct {
	WechatMiniProgram oAuth.WechatMiniProgram
	Account           auth.Account
	Phone             auth.Phone
	IdentityService   identity.IdentityService
	ProfileService    profile.ProfileService
	TokenService      token.TokenService
	UserService       user.UserService
	QueryService      userList.QueryService
	CodeRepo          redis.CodeRepo
}

// SetupServices 初始化仓库层和服务层
// 输入: deps *main.AppDependencies，提供基础依赖
// 输出: *AppServices，包含所有服务实例
func SetupServices(deps *AppDependencies) *AppServices {
	// 初始化 MySQL 仓库
	identityRepo := mysql.NewIdentityRepository(deps.DB)
	userRepo := mysql.NewUserRepository(deps.DB)
	profileRepo := mysql.NewProfileRepository(deps.DB)
	joinQuery := mysql.NewJoinQuery(deps.DB)

	// 初始化 Redis 仓库
	codeRepo := redis.NewCodeRepo(deps.RedisClient)
	tokenBlackRepo := redis.NewTokenBlacklistRepo(deps.RedisClient)

	// 初始化服务层
	wechatService := oAuth.NewWechatMiniProgram(
		identityRepo,
		userRepo,
		tokenBlackRepo,
		deps.JwtToken,
		deps.WechatClient,
		deps.DB,
	)

	accountService := auth.NewAccount(
		identityRepo,
		userRepo,
		tokenBlackRepo,
		deps.JwtToken,
		deps.DB,
	)

	phoneService := auth.NewPhone(
		identityRepo,
		userRepo,
		codeRepo,
		deps.JwtToken,
		deps.DB,
	)

	identityService := identity.NewIdentityService(identityRepo)
	profileService := profile.NewProfileService(profileRepo)
	tokenService := token.NewTokenService(tokenBlackRepo, userRepo, deps.JwtToken)
	userService := user.NewUserService(userRepo)
	queryService := userList.NewQueryService(joinQuery)

	// 封装服务实例
	return &AppServices{
		WechatMiniProgram: wechatService,
		Account:           accountService,
		Phone:             phoneService,
		IdentityService:   identityService,
		ProfileService:    profileService,
		TokenService:      tokenService,
		UserService:       userService,
		QueryService:      queryService,
		CodeRepo:          codeRepo,
	}
}
