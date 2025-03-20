package userError

import "errors"

//var (
//	// 用户相关错误
//
//	ErrUserDisabled     = errors.New("用户已禁用")
//	ErrAccountExists    = errors.New("账号已存在")
//	ErrUserCreateFailed = errors.New("用户创建失败")
//
//	// 认证相关错误
//
//	ErrInvalidToken = errors.New("无效的token")
//	ErrInvalidAuth  = errors.New("用户名或密码错误")
//	ErrAccessDenied = errors.New("权限不足")
//
//	//  数据相关错误
//
//	ErrInvalidDataFormat = errors.New("数据格式错误")
//)

var ErrUserNotFound = errors.New("用户不存在")

var ErrProfileNotFound = errors.New("用户资料不存在")

var ErrAccountAlreadyExists = errors.New("该账号已存在")

var ErrUserNotRegistered = errors.New("该用户未注册")

var ErrPasswordIncorrect = errors.New("密码错误")

var ErrServerInternal = errors.New("服务器内部错误")

var ErrIdentityNotFound = errors.New("身份未找到")

var ErrTokenBlacklistFailed = errors.New("退出登录失败")

var ErrInvalidRefreshToken = errors.New("刷新令牌已过期")

var ErrRefreshTokenExpired = errors.New("刷新令牌已失效")

var ErrCaptchaExpired = errors.New("验证码过期或不存在")

var ErrCaptchaInvalid = errors.New("验证码错误")
