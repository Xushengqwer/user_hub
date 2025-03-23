package user

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"user_hub/models/dto"
	"user_hub/models/entities"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/userError"
)

// UserService 定义用户管理服务接口
type UserService interface {
	// CreateUser 创建新用户
	// - 输入: ctx 上下文, dto 创建用户 DTO
	// - 输出: *vo.UserVO 用户响应, error 操作错误
	CreateUser(ctx context.Context, dto *dto.CreateUserDTO) (*vo.UserVO, error)

	// GetUserByID 根据用户 ID 获取用户信息
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *vo.UserVO 用户响应, error 操作错误
	GetUserByID(ctx context.Context, userID string) (*vo.UserVO, error)

	// UpdateUser 更新用户信息
	// - 输入: ctx 上下文, userID 用户 ID, dto 更新用户 DTO
	// - 输出: *vo.UserVO 用户响应, error 操作错误
	UpdateUser(ctx context.Context, userID string, dto *dto.UpdateUserDTO) (*vo.UserVO, error)

	// DeleteUser 删除用户（软删除）
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *vo.DeleteUserVO 删除响应, error 操作错误
	DeleteUser(ctx context.Context, userID string) error

	// BlackUser 拉黑用户
	// - 输入: ctx 上下文, userID 用户 ID
	// - 输出: *vo.BlackUserVO 拉黑响应, error 操作错误
	BlackUser(ctx context.Context, userID string) error
}

// userService 实现 UserService 接口的结构体
type userService struct {
	repo mysql.UserRepository // 用户仓库实例，用于数据库操作
}

// NewUserService 创建 UserService 实例
// - 输入: repo 用户仓库实例
// - 输出: UserService 接口实现
func NewUserService(repo mysql.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

// CreateUser 创建新用户
// - 输入: ctx 上下文, dto 创建用户 DTO
// - 输出: *vo.UserVO 用户响应, error 操作错误
func (s *userService) CreateUser(ctx context.Context, dto *dto.CreateUserDTO) (*vo.UserVO, error) {
	// 1. 生成用户 ID
	// - 使用 UUID v4 生成用户的唯一标识
	userID := uuid.New().String()

	// 2. 创建用户实体
	// - 根据 DTO 初始化用户角色和状态
	user := &entities.User{
		UserID:   userID,
		UserRole: dto.UserRole,
		Status:   dto.Status,
	}

	// 3. 调用仓库层创建用户
	// - 将用户实体插入数据库
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// 4. 构造并返回用户视图对象
	// - 将实体转换为 VO，返回给调用方
	userVO := &vo.UserVO{
		UserID:    user.UserID,
		UserRole:  user.UserRole,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return userVO, nil
}

// GetUserByID 根据用户 ID 获取用户信息
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: *vo.UserVO 用户响应, error 操作错误
func (s *userService) GetUserByID(ctx context.Context, userID string) (*vo.UserVO, error) {
	// 1. 调用仓库层查询用户
	// - 根据用户 ID 获取用户实体
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			return nil, err
		}
		return nil, err
	}

	// 2. 构造并返回用户视图对象
	// - 将实体转换为 VO，返回给调用方
	userVO := &vo.UserVO{
		UserID:    user.UserID,
		UserRole:  user.UserRole,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return userVO, nil
}

// UpdateUser 更新用户信息
// - 输入: ctx 上下文, userID 用户 ID, dto 更新用户 DTO
// - 输出: *vo.UserVO 用户响应, error 操作错误
func (s *userService) UpdateUser(ctx context.Context, userID string, dto *dto.UpdateUserDTO) (*vo.UserVO, error) {
	// 1. 查询用户是否存在
	// - 根据用户 ID 获取用户实体
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userError.ErrUserNotFound) {
			return nil, err
		}
		return nil, err
	}

	// 2. 更新用户字段
	// - 根据 DTO 更新角色和状态（仅更新非零值）
	if dto.UserRole != 0 {
		user.UserRole = dto.UserRole
	}
	if dto.Status != 0 {
		user.Status = dto.Status
	}

	// 3. 调用仓库层更新用户
	// - 将更新后的实体保存到数据库
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// 4. 构造并返回用户视图对象
	// - 将更新后的实体转换为 VO，返回给调用方
	userVO := &vo.UserVO{
		UserID:    user.UserID,
		UserRole:  user.UserRole,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return userVO, nil
}

// DeleteUser 删除用户（软删除）
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
func (s *userService) DeleteUser(ctx context.Context, userID string) error {
	// 1. 调用仓库层执行软删除
	// - 将用户标记为已删除
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		return err
	}

	// 2. 返回 nil 表示成功
	// - 控制器层将通过 RespondSuccess 返回成功消息，例如 "用户删除成功"
	return nil
}

// BlackUser 拉黑用户
// - 输入: ctx 上下文, userID 用户 ID
// - 输出: error 操作错误
func (s *userService) BlackUser(ctx context.Context, userID string) error {
	// 1. 调用仓库层拉黑用户
	// - 更新用户状态为黑名单
	if err := s.repo.BlackUser(ctx, userID); err != nil {
		return err
	}

	// 2. 返回 nil 表示成功
	// - 控制器层将通过 RespondSuccess 返回成功消息，例如 "用户已拉黑"
	return nil
}
