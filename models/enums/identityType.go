package enums

// IdentityType 身份类型枚举
type IdentityType uint

const (
	AccountPassword   IdentityType = 0 // 账号密码（网站）
	WechatMiniProgram IdentityType = 1 // 微信（小程序）
	Phone             IdentityType = 2 // 手机号（APP）
	// 可扩展其他类型，如 Email、AppleID 等
)
