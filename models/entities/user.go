package entities

import (
	"github.com/Xushengqwer/go-common/models/enums"
	"gorm.io/gorm"
	"time"
)

// User 用户核心信息
type User struct {
	// 用户ID，使用 UUID 作为主键
	UserID string `gorm:"type:char(36);primary_key"`

	// 用户角色（0=游客, 1=用户, 2=管理员），默认值为 0
	UserRole enums.UserRole `gorm:"type:int;default:0"`

	// 用户状态（0=活跃, 1=冻结, 2=注销），默认值为 0
	Status enums.UserStatus `gorm:"type:int;default:0"`

	// 创建时间，默认当前时间戳
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`

	// 更新时间，默认当前时间戳，自动更新
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`

	// 软删除时间戳，列名为 deleted_at
	DeletedAt gorm.DeletedAt `gorm:"type:timestamp;column:deleted_at"`
}
