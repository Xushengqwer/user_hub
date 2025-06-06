package dto

import "time"

// UserQueryDTO 定义用户查询请求结构体
// - 用于管理员分页查询用户及其资料信息
type UserQueryDTO struct {
	// 精确匹配条件（如 user_id="123", status=0）
	Filters map[string]interface{} `json:"filters" binding:"omitempty" `
	// 模糊匹配条件（如 username LIKE "%test%"）
	LikeFilters map[string]string `json:"like_filters" binding:"omitempty" example:"{\"username\": \"test\"}"`
	// 时间范围条件（如 created_at 在某个范围内）
	TimeRangeFilters map[string][2]time.Time `json:"time_range_filters" binding:"omitempty" `
	// 排序字段（如 "created_at DESC"）
	OrderBy string `json:"order_by" binding:"omitempty" example:"created_at DESC"`
	// 页码，默认 1
	Page int `json:"page" binding:"gte=1" example:"1"`
	// 每页大小，默认 10
	PageSize int `json:"page_size" binding:"gte=1,lte=100" example:"10"`
}
