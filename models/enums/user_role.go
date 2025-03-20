package enums

// UserRole 用户角色枚举
type UserRole uint

const (
	Admin UserRole = iota // 0 - 管理员
	User                  // 1 - 普通用户
	Guest                 // 2 - 客人
)
