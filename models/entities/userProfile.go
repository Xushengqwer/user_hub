package entities

import (
	"github.com/Xushengqwer/user_hub/models/enums"
	"time"
)

// UserProfile 用户资料信息
type UserProfile struct {
	//主键ID
	ID uint `gorm:"primary_key;auto_increment"`

	// 关联 User 表的 UserID，外键+级联删除
	UserID string `gorm:"type:char(36);not null;index;foreignKey:UserID;references:user_id;constraint:OnDelete:CASCADE"`

	// 昵称
	Nickname string `gorm:"type:varchar(255)"`

	// 头像 URL
	AvatarURL string `gorm:"type:varchar(255)"`

	// 性别 (0=未知, 1=男, 2=女)，默认值为 0
	Gender enums.Gender `gorm:"type:int;default:0"`

	// 省份
	Province string `gorm:"type:varchar(255)"`

	// 城市
	City string `gorm:"type:varchar(255)"`

	// 创建时间，默认当前时间戳
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`

	// 更新时间，默认当前时间戳，自动更新
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`
}
