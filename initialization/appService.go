package initialization

import (
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/service/userManage"

	// 导入重构后的 service 包路径 (根据实际路径调整)
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/repository/redis"
	"github.com/Xushengqwer/user_hub/service/identity"
	"github.com/Xushengqwer/user_hub/service/login/auth"
	"github.com/Xushengqwer/user_hub/service/login/oAuth"
	"github.com/Xushengqwer/user_hub/service/profile" // 确保导入 profile 服务
	"github.com/Xushengqwer/user_hub/service/token"
	"github.com/Xushengqwer/user_hub/service/userList"
)

// AppServices 封装了应用所需的所有服务层实例。
// ... (结构体定义保持不变)
type AppServices struct {
	WechatMiniProgram oAuth.WechatMiniProgramService
	Account           auth.AccountService
	Phone             auth.PhoneAuthService
	IdentityService   identity.UserIdentityService
	ProfileService    profile.UserProfileService // 这个字段应该已经存在
	TokenService      token.AuthTokenService
	UserService       userManage.UserManageService
	QueryService      userList.UserListQueryService
	CodeRepo          redis.CodeRepo
	SMS               dependencies.SMSClient
}

// SetupServices 初始化所有仓库层和服务层实例。
func SetupServices(deps *AppDependencies) *AppServices {
	// 1. 初始化 MySQL 仓库实例 (这部分保持不变)
	identityRepo := mysql.NewIdentityRepository(deps.DB)
	userRepo := mysql.NewUserRepository(deps.DB)
	profileRepo := mysql.NewProfileRepository(deps.DB)
	joinQuery := mysql.NewJoinQuery(deps.DB)

	// 2. 初始化 Redis 仓库实例 (这部分保持不变)
	codeRepo := redis.NewCodeRepo(deps.RedisClient)
	tokenBlackRepo := redis.NewTokenBlacklistRepo(deps.RedisClient)

	// 3. 初始化服务层实例

	// 首先初始化 UserProfileService，因为它会被其他服务依赖
	profileService := profile.NewUserProfileService(
		userRepo,
		profileRepo,
		deps.DB,
		deps.Logger,
		deps.COSClient,
	)

	// 初始化微信小程序认证服务，并注入 profileService
	wechatService := oAuth.NewWechatMiniProgramService(
		identityRepo,
		userRepo,
		profileRepo,
		tokenBlackRepo,
		deps.JwtToken,
		deps.WechatClient,
		deps.DB,
		deps.Logger,
	)

	// 初始化账号密码认证服务，并注入 profileService
	accountService := auth.NewAccountService(
		identityRepo,
		userRepo,
		profileRepo,
		tokenBlackRepo,
		deps.JwtToken,
		deps.DB,
		deps.Logger,
	)

	// 初始化手机号认证服务，并注入 profileService
	phoneService := auth.NewPhoneAuthService(
		identityRepo,
		userRepo,
		profileRepo,
		codeRepo,
		deps.JwtToken,
		deps.DB,
		deps.Logger,
	)

	// 初始化其他服务 (保持不变)
	identityService := identity.NewUserIdentityService(
		identityRepo,
		deps.DB,
		deps.Logger,
	)

	tokenService := token.NewAuthTokenService(
		tokenBlackRepo,
		userRepo,
		deps.JwtToken,
		deps.Logger,
	)

	userService := userManage.NewUserService(
		userRepo,
		identityRepo,
		profileRepo, // UserManageService 也可能需要 profileRepo (例如，如果它也创建用户配置文件)
		deps.DB,
		deps.Logger,
		// 如果 UserManageService.CreateUser 也需要创建 profile,
		// 那么它也需要 profileService。
		// profileService, // <-- 如果 UserManageService 需要，则取消此行注释
	)

	queryService := userList.NewUserListQueryService(
		joinQuery,
		deps.Logger,
	)

	// 4. 封装所有初始化完成的服务实例到 AppServices 结构体中
	return &AppServices{
		WechatMiniProgram: wechatService,
		Account:           accountService,
		Phone:             phoneService,
		IdentityService:   identityService,
		ProfileService:    profileService, // 确保 profileService 被正确赋值
		TokenService:      tokenService,
		UserService:       userService,
		QueryService:      queryService,
		CodeRepo:          codeRepo,
		SMS:               deps.SMSClient,
	}
}
