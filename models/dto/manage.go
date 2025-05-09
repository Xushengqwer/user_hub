package dto

import "github.com/Xushengqwer/go-common/models/enums"

// CreateUserDTO 定义创建用户的数据传输对象
// - 用于管理员创建新用户时的请求体
type CreateUserDTO struct {
	// 用户角色（0=管理员, 1=普通用户, 2=客人）
	// - 必填字段，验证角色枚举值
	UserRole enums.UserRole `json:"user_role" binding:"oneof=0 1 2"`

	// 用户状态（0=活跃, 1=拉黑）
	// - 必填字段，验证状态枚举值
	Status enums.UserStatus `json:"status" binding:"oneof=0 1"`
}

// UpdateUserDTO 定义更新用户请求结构体
// - 用于管理员更新用户角色和状态
type UpdateUserDTO struct {
	// 用户角色（0=Admin, 1=User, 2=Guest），可选
	UserRole enums.UserRole `json:"user_role" binding:"omitempty,oneof=0 1 2" example:"1"`
	// 用户状态（0=Active, 1=Blacklisted），可选
	Status enums.UserStatus `json:"status" binding:"omitempty,oneof=0 1" example:"0"`
}
