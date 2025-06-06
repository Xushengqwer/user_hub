package vo

import (
	commonEnums "github.com/Xushengqwer/go-common/models/enums" // 公共枚举
	projectEnums "github.com/Xushengqwer/user_hub/models/enums" // 项目内部枚举
	"time"
)

type MyAccountDetailVO struct {
	UserID    string                 `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserRole  commonEnums.UserRole   `json:"user_role" example:"1"` // 来自 User 实体
	Status    commonEnums.UserStatus `json:"status" example:"0"`    // 来自 User 实体
	Nickname  string                 `json:"nickname" example:"小明"` // 来自 UserProfile 实体
	AvatarURL string                 `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	Gender    projectEnums.Gender    `json:"gender" example:"1"`
	Province  string                 `json:"province" example:"广东"`
	City      string                 `json:"city" example:"深圳"`
	CreatedAt time.Time              `json:"created_at" example:"2023-01-01T00:00:00Z"` // 可以是 User 的创建时间
	UpdatedAt time.Time              `json:"updated_at" example:"2023-01-01T00:00:00Z"` // 可以是 User 或 Profile 中较新的更新时间
}
