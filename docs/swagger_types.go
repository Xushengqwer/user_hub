package docs

// 这个文件定义了专门用于 Swagger 文档注解的类型。
// 由于 swaggo/swag 工具目前不支持直接解析泛型类型（如 response.APIResponse[T]），
// 我们需要为每个在控制器注解中使用的具体泛型实例化类型定义一个非泛型的包装器。

import (
	"github.com/Xushengqwer/go-common/response" // 导入您的通用响应包
	"github.com/Xushengqwer/user_hub/models/vo" // 导入您的 VO 包
	// 如果需要，导入其他包，例如 enums
)

// --- 成功响应包装类型 ---

// SwaggerAPIUserinfoResponse 包装了 response.APIResponse[vo.Userinfo]
// 用于 AccountController.RegisterHandler
type SwaggerAPIUserinfoResponse struct {
	response.APIResponse[vo.Userinfo]
}

// SwaggerAPILoginResponse 包装了 response.APIResponse[vo.LoginResponse]
// 用于 AccountController.LoginHandler, PhoneAuthController.LoginOrRegisterHandler, WechatAuthController.LoginOrRegisterHandler
type SwaggerAPILoginResponse struct {
	response.APIResponse[vo.LoginResponse]
}

// SwaggerAPIMyAccountDetailResponse 包装了 response.APIResponse[vo.MyAccountDetailVO]
// 用于获取登录用户的个人信息给前端展示
type SwaggerAPIMyAccountDetailResponse struct {
	response.APIResponse[vo.MyAccountDetailVO]
}

// SwaggerAPIEmptyResponse 包装了 response.APIResponse[vo.Empty] (用于表示成功但无数据返回的情况)
// 用于 AuthController.SendCaptcha, IdentityController.DeleteIdentityHandler,
// TokenController.Logout, UserController.DeleteUserHandler, UserController.BlackUserHandler,
// UserProfileController.DeleteProfileHandler
type SwaggerAPIEmptyResponse struct {
	response.APIResponse[vo.Empty]
}

// SwaggerAPITokenPairResponse 包装了 response.APIResponse[vo.TokenPair]
// 用于 TokenController.RefreshToken
type SwaggerAPITokenPairResponse struct {
	response.APIResponse[vo.TokenPair]
}

// SwaggerAPIIdentityVOResponse 包装了 response.APIResponse[vo.IdentityVO]
// 用于 IdentityController.CreateIdentityHandler, IdentityController.UpdateIdentityHandler
type SwaggerAPIIdentityVOResponse struct {
	response.APIResponse[vo.IdentityVO]
}

// SwaggerAPIIdentityListResponse 包装了 response.APIResponse[vo.IdentityList]
// 用于 IdentityController.GetIdentitiesByUserIDHandler
type SwaggerAPIIdentityListResponse struct {
	response.APIResponse[vo.IdentityList]
}

// SwaggerAPIIdentityTypeListResponse 包装了 response.APIResponse[vo.IdentityTypeList]
// 用于 IdentityController.GetIdentityTypesByUserIDHandler
type SwaggerAPIIdentityTypeListResponse struct {
	response.APIResponse[vo.IdentityTypeList]
}

// SwaggerAPIProfileVOResponse 包装了 response.APIResponse[vo.ProfileVO]
// 用于 UserProfileController.CreateProfileHandler, UserProfileController.GetProfileByUserIDHandler,
// UserProfileController.UpdateProfileHandler
type SwaggerAPIProfileVOResponse struct {
	response.APIResponse[vo.ProfileVO]
}

// SwaggerAPIUserVOResponse 包装了 response.APIResponse[vo.UserVO]
// 用于 UserController.CreateUserHandler, UserController.GetUserByIDHandler, UserController.UpdateUserHandler
type SwaggerAPIUserVOResponse struct {
	response.APIResponse[vo.UserVO]
}

// SwaggerAPIUserListResponse 包装了 response.APIResponse[vo.UserListResponse]
// 用于 UserListQueryController.ListUsersWithProfileHandler
type SwaggerAPIUserListResponse struct {
	response.APIResponse[vo.UserListResponse]
}

// --- 失败响应包装类型 ---

// SwaggerAPIErrorResponseString 包装了 response.APIResponse[string]
type SwaggerAPIErrorResponseString struct {
	response.APIResponse[string]
}

// SwaggerAPIErrorResponseAny 包装了 response.APIResponse[any]
type SwaggerAPIErrorResponseAny struct {
	response.APIResponse[any]
}

// --- 您可能需要的其他包装类型 ---
// 例如，如果某个接口返回 response.APIResponse[[]*vo.SomeOtherVO]
// type SwaggerAPISomeOtherListResponse struct {
//     response.APIResponse[[]*vo.SomeOtherVO]
// }
