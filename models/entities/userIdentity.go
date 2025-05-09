package entities

import (
	"github.com/Xushengqwer/user_hub/models/enums"
	"time"
)

// UserIdentity 用户身份信息
type UserIdentity struct {
	// 自增主键
	IdentityID uint `gorm:"primary_key;auto_increment"`

	// 关联 User 表的 UserID，外键
	UserID string `gorm:"type:char(36);not null;index;foreignKey:UserID;references:user_id;constraint:OnDelete:CASCADE"`

	// 身份类型（0=账号密码, 1=小程序, 2=手机号）
	IdentityType enums.IdentityType `gorm:"type:int;not null"`

	// 标识符，如账号、OpenID、手机号，具有唯一性索引
	Identifier string `gorm:"type:varchar(255);not null;uniqueIndex:idx_type_identifier"`

	// 凭证，如密码（哈希）、UnionID
	Credential string `gorm:"type:varchar(255)"`

	// 创建时间，默认当前时间戳
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`

	// 更新时间，默认当前时间戳，自动更新
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`
}
