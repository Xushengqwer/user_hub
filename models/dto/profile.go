package dto

import "user_hub/models/enums"

// CreateProfileDTO 定义创建资料请求结构体
// - 用于用户首次填写个人资料时接收请求数据
type CreateProfileDTO struct {
	// 用户 ID
	UserID string `json:"user_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 昵称
	Nickname string `json:"nickname" binding:"omitempty" example:"小明"`
	// 头像 URL
	AvatarURL string `json:"avatar_url" binding:"omitempty,url" example:"https://example.com/avatar.jpg"`
	// 性别（0=未知, 1=男, 2=女）
	Gender enums.Gender `json:"gender" binding:"omitempty,oneof=0 1 2" example:"1"`
	// 省份
	Province string `json:"province" binding:"omitempty" example:"广东"`
	// 城市
	City string `json:"city" binding:"omitempty" example:"深圳"`
}

// UpdateProfileDTO 定义更新资料请求结构体
// - 用于用户或管理员更新资料时接收请求数据
type UpdateProfileDTO struct {
	// 昵称
	Nickname string `json:"nickname" binding:"omitempty" example:"小明"`
	// 头像 URL
	AvatarURL string `json:"avatar_url" binding:"omitempty,url" example:"https://example.com/avatar.jpg"`
	// 性别（0=未知, 1=男, 2=女）
	Gender enums.Gender `json:"gender" binding:"omitempty,oneof=0 1 2" example:"1"`
	// 省份
	Province string `json:"province" binding:"omitempty" example:"广东"`
	// 城市
	City string `json:"city" binding:"omitempty" example:"深圳"`
}
