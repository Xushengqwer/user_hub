package vo

import (
	"github.com/Xushengqwer/go-common/models/enums"
	"time"
)

// UserVO 定义用户响应结构体
// - 用于返回用户信息
type UserVO struct {
	// 用户 ID
	UserID string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 用户角色（0=Admin, 1=User, 2=Guest）
	UserRole enums.UserRole `json:"user_role" example:"1"`
	// 用户状态（0=Active, 1=Blacklisted）
	Status enums.UserStatus `json:"status" example:"0"`
	// 创建时间
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}
