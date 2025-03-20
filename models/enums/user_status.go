package enums

// UserStatus 定义用户状态枚举
type UserStatus uint

const (
	Active      UserStatus = 0 // 活跃
	Blacklisted UserStatus = 1 // 拉黑
)
