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
	"github.com/Xushengqwer/user_hub/service/profile"
	"github.com/Xushengqwer/user_hub/service/token"
	"github.com/Xushengqwer/user_hub/service/userList"
)

// AppServices 封装了应用所需的所有服务层实例。
// 设计目的:
//   - 提供一个集中的地方来管理和访问所有服务。
//   - 简化在应用启动时（如 main.go 或 router.go）传递依赖的过程。
type AppServices struct {
	WechatMiniProgram oAuth.WechatMiniProgramService // 使用更新后的接口名
	Account           auth.AccountService            // 使用更新后的接口名
	Phone             auth.PhoneAuthService          // 使用更新后的接口名
	IdentityService   identity.UserIdentityService   // 使用更新后的接口名
	ProfileService    profile.UserProfileService     // 使用更新后的接口名
	TokenService      token.AuthTokenService         // 使用更新后的接口名
	UserService       userManage.UserManageService   // 保持不变 (如果未重命名)
	QueryService      userList.UserListQueryService  // 使用更新后的接口名
	CodeRepo          redis.CodeRepo                 // CodeRepo 通常直接暴露给需要的服务或控制器，这里保留以便 AuthController 使用
	SMS               dependencies.SMSClient         // SMSClient 也保留，供 AuthController 使用
}

// SetupServices 初始化所有仓库层和服务层实例。
// 设计目的:
//   - 执行依赖注入，将底层的仓库实例、配置、工具（如 logger, jwtUtil）和数据库连接注入到相应的服务层实现中。
//   - 返回一个包含所有初始化完成的服务实例的 AppServices 结构体。
//
// 参数:
//   - deps: *AppDependencies 结构体指针，包含了所有基础依赖项（如 DB 连接、Redis 客户端、Logger、配置等）。
//
// 返回:
//   - *AppServices: 包含所有服务实例的结构体指针。
func SetupServices(deps *AppDependencies) *AppServices {
	// 1. 初始化 MySQL 仓库实例
	//    每个仓库接收原始的数据库连接 deps.DB。
	//    仓库方法内部根据需要使用 originalDB 或传入的事务 tx。
	identityRepo := mysql.NewIdentityRepository(deps.DB)
	userRepo := mysql.NewUserRepository(deps.DB)
	profileRepo := mysql.NewProfileRepository(deps.DB)
	joinQuery := mysql.NewJoinQuery(deps.DB)

	// 2. 初始化 Redis 仓库实例
	//    依赖 Redis 客户端 deps.RedisClient。
	codeRepo := redis.NewCodeRepo(deps.RedisClient)
	tokenBlackRepo := redis.NewTokenBlacklistRepo(deps.RedisClient) // 使用基于 JTI 的实现

	// 3. 初始化服务层实例 (使用更新后的构造函数和依赖)
	//    注意：为每个服务注入所需的仓库、工具、DB连接和Logger。
	wechatService := oAuth.NewWechatMiniProgramService( // 使用更新后的构造函数名
		identityRepo,
		userRepo,
		tokenBlackRepo, // 即使不用也注入，保持一致性
		deps.JwtToken,
		deps.WechatClient,
		deps.DB,     // 注入 DB 用于事务
		deps.Logger, // 注入 Logger
	)

	accountService := auth.NewAccountService( // 使用更新后的构造函数名
		identityRepo,
		userRepo,
		tokenBlackRepo, // 即使不用也注入
		deps.JwtToken,
		deps.DB,     // 注入 DB 用于事务
		deps.Logger, // 注入 Logger
	)

	phoneService := auth.NewPhoneAuthService( // 使用更新后的构造函数名
		identityRepo,
		userRepo,
		codeRepo, // Phone 服务需要 codeRepo
		deps.JwtToken,
		deps.DB,     // 注入 DB 用于事务
		deps.Logger, // 注入 Logger
	)

	identityService := identity.NewUserIdentityService( // 使用更新后的构造函数名
		identityRepo,
		deps.DB,     // 注入 DB 用于传递给仓库写操作
		deps.Logger, // 注入 Logger
	)

	profileService := profile.NewUserProfileService( // 使用更新后的构造函数名
		profileRepo,
		deps.DB,     // 注入 DB 用于传递给仓库写操作
		deps.Logger, // 注入 Logger
	)

	tokenService := token.NewAuthTokenService( // 使用更新后的构造函数名
		tokenBlackRepo, // Token 服务需要 tokenBlackRepo
		userRepo,       // 需要 userRepo 获取用户信息
		deps.JwtToken,
		deps.Logger, // 注入 Logger
	)

	userService := userManage.NewUserService( // 使用更新后的构造函数名
		userRepo,
		identityRepo, // UserService 的 DeleteUser 需要 identityRepo
		profileRepo,  // UserService 的 DeleteUser 需要 profileRepo
		deps.DB,      // 注入 DB 用于事务和传递
		deps.Logger,  // 注入 Logger
	)

	queryService := userList.NewUserListQueryService( // 使用更新后的构造函数名
		joinQuery,   // Query 服务需要 joinQuery 仓库
		deps.Logger, // 注入 Logger
		// deps.DB, // 通常不需要 DB，除非 JoinQuery 实现需要外部事务控制
	)

	// 4. 封装所有初始化完成的服务实例到 AppServices 结构体中
	return &AppServices{
		WechatMiniProgram: wechatService,
		Account:           accountService,
		Phone:             phoneService,
		IdentityService:   identityService,
		ProfileService:    profileService,
		TokenService:      tokenService,
		UserService:       userService,
		QueryService:      queryService,
		CodeRepo:          codeRepo,       // 将 codeRepo 也放入，以便 AuthController 使用
		SMS:               deps.SMSClient, // 将 SMSClient 也放入，以便 AuthController 使用
	}
}
