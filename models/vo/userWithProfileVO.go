package vo

import (
	"github.com/Xushengqwer/go-common/models/enums"
	myenums "github.com/Xushengqwer/user_hub/models/enums"
	"time"
)

// UserWithProfileVO 定义用户及其资料响应结构体
// - 用于返回用户和资料的联合查询结果
type UserWithProfileVO struct {
	// 用户 ID
	UserID string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 用户角色（0=Admin, 1=User, 2=Guest）
	Role enums.UserRole `json:"role" example:"1"`
	// 用户状态（0=Active, 1=Blacklisted）
	Status enums.UserStatus `json:"status" example:"0"`
	// 昵称
	Nickname string `json:"nickname" example:"小明"`
	// 头像 URL
	AvatarURL string `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	// 性别（0=未知, 1=男, 2=女）
	Gender myenums.Gender `json:"gender" example:"1"`
	// 省份
	Province string `json:"province" example:"广东"`
	// 城市
	City string `json:"city" example:"深圳"`
	// 创建时间
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

type UserListResponse struct {
	Users []*UserWithProfileVO `json:"users"`
	Total int64                `json:"total"`
}
