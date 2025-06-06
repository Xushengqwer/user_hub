package mysql

import (
	"context"
	"fmt" // 引入 fmt 包用于错误包装
	"strings"

	"github.com/Xushengqwer/user_hub/models/dto" // 引入 DTO 包
	"github.com/Xushengqwer/user_hub/models/vo"  // 引入 VO 包

	"gorm.io/gorm"
)

// 定义允许的过滤字段及其对应的数据库列名
// Key: DTO 中的字段名 (客户端传入), Value: 数据库中的安全列名 (带表前缀)
var allowedFilters = map[string]string{
	"status":     "users.status",
	"nickname":   "user_profiles.nickname", // 假设允许按昵称过滤
	"created_at": "users.created_at",       // 用于时间范围
	// ... 在这里添加其他允许过滤的字段
}

// 定义允许的排序字段及其对应的数据库列名
var allowedOrderBy = map[string]string{
	"created_at": "users.created_at",
	"user_id":    "users.user_id",
	// ... 在这里添加其他允许排序的字段
}

// JoinQuery 定义了专注于多表联合查询的操作接口。
// - 它提供了比单个实体仓库更复杂的查询能力。
type JoinQuery interface {
	// ListUsersWithProfile 分页查询用户及其关联的资料信息。
	// - 直接接收服务层传递的查询 DTO，并对其进行安全处理。
	// - 直接返回用于 API 响应的 VO 列表，减少服务层的转换工作。
	// - 如果数据库查询失败，则返回包装后的错误。
	ListUsersWithProfile(ctx context.Context, queryDTO *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error)
}

// joinQuery 是 JoinQuery 接口基于 GORM 的实现。
type joinQuery struct {
	db *gorm.DB // db 是 GORM 数据库连接实例
}

// NewJoinQuery 创建一个新的 joinQuery 实例。
// - 依赖注入 GORM 数据库连接。
func NewJoinQuery(db *gorm.DB) JoinQuery {
	return &joinQuery{db: db}
}

// ListUsersWithProfile 实现接口方法，执行用户与资料的联合分页查询，并安全处理 DTO 输入。
func (r *joinQuery) ListUsersWithProfile(ctx context.Context, queryDTO *dto.UserQueryDTO) ([]*vo.UserWithProfileVO, int64, error) {
	var results []*vo.UserWithProfileVO

	// 1. 构建基础查询 (与之前相同)
	db := r.db.WithContext(ctx).
		Table("users").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Select("users.user_id, users.user_role as role, users.status, " +
			"user_profiles.nickname, user_profiles.avatar_url, user_profiles.gender, " +
			"user_profiles.province, user_profiles.city, " +
			"users.created_at, users.updated_at")

	// 2. 安全地应用过滤条件
	// - 精确匹配
	if queryDTO.Filters != nil {
		for key, value := range queryDTO.Filters {
			// 验证 key 是否是允许的过滤字段
			if dbColumn, ok := allowedFilters[key]; ok {
				// 使用映射后的安全数据库列名
				db = db.Where(dbColumn+" = ?", value)
			} else {
				// 忽略或记录未允许的过滤键 (或者返回错误，取决于策略)
				fmt.Printf("警告: 忽略了不允许的过滤键: %s\n", key) // 示例：打印警告
			}
		}
	}
	// - 模糊匹配
	if queryDTO.LikeFilters != nil {
		for key, value := range queryDTO.LikeFilters {
			// 验证 key 是否是允许的过滤字段
			if dbColumn, ok := allowedFilters[key]; ok {
				// 使用映射后的安全数据库列名
				db = db.Where(dbColumn+" LIKE ?", "%"+value+"%")
			} else {
				fmt.Printf("警告: 忽略了不允许的模糊过滤键: %s\n", key)
			}
		}
	}
	// - 时间范围
	if queryDTO.TimeRangeFilters != nil {
		for key, times := range queryDTO.TimeRangeFilters {
			// 验证 key 是否是允许的过滤字段
			if dbColumn, ok := allowedFilters[key]; ok {
				if !times[0].IsZero() {
					db = db.Where(dbColumn+" >= ?", times[0])
				}
				if !times[1].IsZero() {
					db = db.Where(dbColumn+" <= ?", times[1])
				}
			} else {
				fmt.Printf("警告: 忽略了不允许的时间范围过滤键: %s\n", key)
			}
		}
	}

	// 3. 获取总记录数 (在应用分页和排序之前)
	countDb := db // 创建副本用于 Count
	var total int64
	if err := countDb.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("joinQuery.ListUsersWithProfile: 查询总数失败: %w", err)
	}

	// 4. 安全地应用排序
	orderByClause := "users.created_at DESC" // 默认排序
	if queryDTO.OrderBy != "" {
		parts := strings.Fields(queryDTO.OrderBy) // 按空格分割，例如 "created_at DESC"
		field := parts[0]
		direction := "ASC" // 默认升序
		if len(parts) > 1 {
			dirUpper := strings.ToUpper(parts[1])
			if dirUpper == "DESC" {
				direction = "DESC"
			} else if dirUpper != "ASC" {
				// 如果方向不是 ASC 或 DESC，则忽略或报错
				fmt.Printf("警告: 忽略了无效的排序方向: %s\n", parts[1])
				direction = "" // 标记为无效，使用默认排序
			}
		}

		// 验证排序字段是否允许
		if dbColumn, ok := allowedOrderBy[field]; ok && direction != "" {
			orderByClause = dbColumn + " " + direction
		} else {
			fmt.Printf("警告: 忽略了不允许或无效的排序字段: %s\n", field)
			// 使用默认排序
		}
	}
	db = db.Order(orderByClause)

	// 5. 应用分页 (与之前相同)
	page := queryDTO.Page
	pageSize := queryDTO.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	db = db.Offset(offset).Limit(pageSize)

	// 6. 执行最终查询 (与之前相同)
	if err := db.Scan(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("joinQuery.ListUsersWithProfile: 查询用户列表失败: %w", err)
	}

	// 7. 返回结果
	return results, total, nil
}
