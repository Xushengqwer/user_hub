package dto

import "github.com/Xushengqwer/user_hub/models/enums"

// CreateIdentityDTO 定义创建身份请求结构体
// - 用于用户绑定新登录方式时接收请求数据
type CreateIdentityDTO struct {
	// 用户 ID
	UserID string `json:"user_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	// 身份类型（0=账号密码, 1=小程序, 2=手机号）
	IdentityType enums.IdentityType `json:"identity_type" binding:"required" example:"0"`
	// 标识符（如账号、OpenID、手机号）
	Identifier string `json:"identifier" binding:"required" example:"user123"`
	// 凭证（如密码哈希、UnionID）
	Credential string `json:"credential" binding:"required" example:"hashed_password"`
}

// UpdateIdentityDTO 定义更新身份请求结构体
// - 用于用户修改密码等操作
type UpdateIdentityDTO struct {
	// 新凭证（如新密码哈希）
	Credential string `json:"credential" binding:"required" example:"new_hashed_password"`
}

// IdentityCredential 定义身份验证所需的最小字段集结构体
// - 用于返回用户身份凭证的核心信息
type IdentityCredential struct {
	UserID     string `gorm:"column:user_id"`    // 用户 ID
	Credential string `gorm:"column:credential"` // 身份凭证（如密码哈希）
}
