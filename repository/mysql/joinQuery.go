package mysql

import (
	"context"
	"gorm.io/gorm"
)

// JoinQuery 定义联合查询接口，专注于多表查询操作
// - 提供复杂查询功能，如用户和资料的联合查询
type JoinQuery interface {
	// ListUsersWithProfile 分页查询用户及其资料信息
	// - 输入: ctx 上下文, query 用户查询条件
	// - 输出: []*UserWithProfile 查询结果列表, int64 总记录数, error 操作错误
	ListUsersWithProfile(ctx context.Context, query UserQuery) ([]*UserWithProfile, int64, error)
}

// joinQuery 实现 JoinQuery 接口的结构体
type joinQuery struct {
	db *gorm.DB // GORM 数据库实例
}

// NewJoinQuery 创建 JoinQuery 实例
// - 输入: db GORM 数据库实例
// - 输出: JoinQuery 接口实现
func NewJoinQuery(db *gorm.DB) JoinQuery {
	return &joinQuery{db: db}
}

// ListUsersWithProfile 分页查询用户及其资料信息
//   - 输入: ctx 上下文, query 用户查询条件
//   - 输出: []*UserWithProfile 查询结果列表, int64 总记录数, error 操作错误
//   - SQL: SELECT users.user_id, users.role, users.status,
//     user_profiles.nickname, user_profiles.avatar_url, user_profiles.gender,
//     user_profiles.province, user_profiles.city,
//     users.created_at, users.updated_at
//     FROM users LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id
//     [WHERE ...] [ORDER BY ...] LIMIT ? OFFSET ?
func (r *joinQuery) ListUsersWithProfile(ctx context.Context, query UserQuery) ([]*UserWithProfile, int64, error) {
	// 1. 定义结果存储
	// - 用于接收联合查询的结果
	var results []*UserWithProfile

	// 2. 构建基础查询
	// - 联合 users 和 user_profiles 表，使用 LEFT JOIN
	// - 选择指定的字段进行查询
	db := r.db.WithContext(ctx).
		Table("users").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Select("users.user_id, users.role, users.status, " +
			"user_profiles.nickname, user_profiles.avatar_url, user_profiles.gender, " +
			"user_profiles.province, user_profiles.city, " +
			"users.created_at, users.updated_at")

	// 3. 应用过滤条件
	// - 根据 query.Filters 添加精确匹配条件
	if query.Filters != nil {
		for key, value := range query.Filters {
			db = db.Where(key+" = ?", value)
		}
	}
	// - 根据 query.LikeFilters 添加模糊匹配条件
	if query.LikeFilters != nil {
		for key, value := range query.LikeFilters {
			db = db.Where(key+" LIKE ?", "%"+value+"%")
		}
	}
	// - 根据 query.TimeRangeFilters 添加时间范围条件
	if query.TimeRangeFilters != nil {
		for key, times := range query.TimeRangeFilters {
			if !times[0].IsZero() {
				db = db.Where(key+" >= ?", times[0])
			}
			if !times[1].IsZero() {
				db = db.Where(key+" <= ?", times[1])
			}
		}
	}

	// 4. 获取总记录数
	// - 使用 Count 计算符合条件的记录总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 5. 应用排序
	// - 根据 query.OrderBy 设置排序规则
	if query.OrderBy != "" {
		db = db.Order(query.OrderBy)
	}

	// 6. 应用分页
	// - 设置默认值并计算偏移量，实现偏移量分页
	if query.Page < 1 {
		query.Page = 1 // 默认第一页
	}
	if query.PageSize < 1 {
		query.PageSize = 10 // 默认每页10条
	}
	offset := (query.Page - 1) * query.PageSize
	db = db.Offset(offset).Limit(query.PageSize)

	// 7. 执行查询
	// - 将查询结果填充到 results 中
	if err := db.Find(&results).Error; err != nil {
		return nil, 0, err
	}

	// 8. 返回结果
	// - 返回查询结果和总记录数
	return results, total, nil
}
