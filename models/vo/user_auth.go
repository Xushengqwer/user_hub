package vo

import (
	"time"
	"user_hub/models/enums"
)

// CurrentUserResponseVO 登录响应数据
type CurrentUserResponseVO struct {
	UserID    string           `json:"userID"` // 用户ID
	Account   string           `json:"account"`
	Nickname  string           `json:"nickname"`
	Role      enums.UserRole   `json:"role"`   // 用户权限
	Status    enums.UserStatus `json:"status"` // 用户状态
	Gender    enums.Gender     `json:"gender"`
	AvatarURL string           `json:"avatar_url"`
	Province  string           `json:"province"`
	City      string           `json:"city"`
	CreatedAt time.Time        `json:"created_at"` // 创建时间
	UpdatedAt time.Time        `json:"updated_at"` // 更新时间
}

// LoginSuccessResponseVO 用户登录响应
type LoginSuccessResponseVO struct {
	Type        string         `json:"type"`         // 登录类型，如 "account"
	Role        enums.UserRole `json:"role"`         // 当前权限，如 "admin"
	AccessToken string         `json:"access_token"` // 认证令牌
	UserID      string         `json:"userID"`       // 用户ID
}
