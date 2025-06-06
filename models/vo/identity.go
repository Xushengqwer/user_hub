package vo

import (
	"github.com/Xushengqwer/user_hub/models/enums"
	"time"
)

// IdentityVO 定义身份响应结构体
// - 用于返回身份信息
type IdentityVO struct {
	// 身份 ID
	IdentityID uint `json:"identity_id" example:"1"`
	// 用户 ID
	UserID string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 身份类型（0=账号密码, 1=小程序, 2=手机号）
	IdentityType enums.IdentityType `json:"identity_type" example:"0"`
	// 标识符（如账号、OpenID、手机号）
	Identifier string `json:"identifier" example:"user123"`
	// 创建时间
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

type IdentityList struct {
	Items []*IdentityVO `json:"items"`
}

type IdentityTypeList struct {
	Items []enums.IdentityType `json:"items"`
}
