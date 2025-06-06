package dto

import "github.com/Xushengqwer/user_hub/models/enums"

// CreateProfileDTO 定义创建资料请求结构体
// - 用于用户首次填写个人资料时接收请求数据
type CreateProfileDTO struct {
	// 用户 ID
	UserID string `json:"user_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 昵称 (可选)
	Nickname string `json:"nickname" binding:"omitempty" example:"小明"`
	// 头像 URL (可选)
	AvatarURL string `json:"avatar_url" binding:"omitempty,url" example:"https://example.com/avatar.jpg"`
	// 性别（0=未知, 1=男, 2=女）(可选)
	Gender enums.Gender `json:"gender" binding:"omitempty,oneof=0 1 2" example:"1"`
	// 省份 (可选)
	Province string `json:"province" binding:"omitempty" example:"广东"`
	// 城市 (可选)
	City string `json:"city" binding:"omitempty" example:"深圳"`
}

// UpdateProfileDTO 定义更新资料请求结构体
// - 用于用户或管理员更新资料时接收请求数据。
// - 使用指针类型字段，只有当请求中明确提供了某个字段时，对应的值才不为 nil，服务层据此进行更新。
type UpdateProfileDTO struct {
	// 昵称 (可选更新)
	Nickname *string `json:"nickname,omitempty" example:"小明"` // 改为指针 *string
	// 性别（0=未知, 1=男, 2=女）(可选更新)
	Gender *enums.Gender `json:"gender,omitempty" example:"1"` // 改为指针 *enums.Gender, 移除了 oneof (Gin 对指针的 oneof 验证可能不直观，可以在服务层验证)
	// 省份 (可选更新)
	Province *string `json:"province,omitempty" example:"广东"` // 改为指针 *string
	// 城市 (可选更新)
	City *string `json:"city,omitempty" example:"深圳"` // 改为指针 *string
}
