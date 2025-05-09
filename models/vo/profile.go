package vo

import (
	"github.com/Xushengqwer/user_hub/models/enums"
	"time"
)

// ProfileVO 定义资料响应结构体
// - 用于返回用户资料信息
type ProfileVO struct {
	// 用户 ID
	UserID string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 昵称
	Nickname string `json:"nickname" example:"小明"`
	// 头像 URL
	AvatarURL string `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	// 性别（0=未知, 1=男, 2=女）
	Gender enums.Gender `json:"gender" example:"1"`
	// 省份
	Province string `json:"province" example:"广东"`
	// 城市
	City string `json:"city" example:"深圳"`
	// 创建时间
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}
