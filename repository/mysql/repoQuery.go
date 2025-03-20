package mysql

import (
	"time"
	"user_hub/models/enums"
)

// Example:
// 查询第一页，每页10条，状态为1、用户名含"john"、2025年1-3月创建的用户，按创建时间倒序。
//
// query := UserQuery{
//     Page:     1,
//     PageSize: 10,
//     Filters:  map[string]interface{}{"users.status": 1},
//     LikeFilters: map[string]string{"users.username": "john"},
//     TimeRangeFilters: map[string][2]time.Time{
//         "users.created_at": {
//             time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
//             time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
//         },
//     },
//     OrderBy: "users.created_at desc",
// }

// UserQuery 通用的查询条件结构体，用于分页和过滤
type UserQuery struct {
	Page             int                     // 页码，从 1 开始
	PageSize         int                     // 每页大小
	Filters          map[string]interface{}  // 精确匹配条件，键名需带表名，如 "users.role"
	LikeFilters      map[string]string       // 模糊查询条件，键名需带表名，如 "users.username"
	TimeRangeFilters map[string][2]time.Time // 时间范围条件，键名需带表名，如 "users.created_at"
	OrderBy          string                  // 排序字段，带表名，如 "users.created_at desc"
}

// UserWithProfile 联合查询返回的用户完整信息结构体
type UserWithProfile struct {
	UserID    string           `json:"user_id"`    // 用户ID
	Role      enums.UserRole   `json:"role"`       // 用户角色
	Status    enums.UserStatus `json:"status"`     // 用户状态
	Nickname  string           `json:"nickname"`   // 用户昵称
	AvatarURL string           `json:"avatar_url"` // 头像URL
	Gender    enums.Gender     `json:"gender"`
	Province  string           `json:"province"`
	City      string           `json:"city"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
