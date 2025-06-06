package token

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装
	"time"

	// 引入公共模块
	"github.com/Xushengqwer/go-common/commonerrors"
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"github.com/Xushengqwer/go-common/models/enums"
	"go.uber.org/zap" // 引入 zap 用于日志字段

	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/repository/mysql"
	"github.com/Xushengqwer/user_hub/repository/redis"
)

// AuthTokenService 定义了管理认证令牌（Access Token 和 Refresh Token）的服务接口。
// 设计目的:
// - 提供统一的令牌吊销（退出登录）和续期（刷新令牌）功能。
// - 与具体的登录方式（账号密码、微信等）解耦，登录服务只需生成初始令牌，后续管理由本服务负责。
// 使用场景:
// - 用户点击“退出登录”按钮。
// - 客户端在 Access Token 过期后，使用 Refresh Token 请求新的令牌对。
type AuthTokenService interface {
	// Logout 处理用户退出登录的请求。
	// 主要逻辑: 解析传入的令牌（通常是 Refresh Token 或 Access Token），获取其 JTI，
	// 并将该 JTI 加入 Redis 黑名单，使其在剩余有效期内失效。
	// 参数:
	//  - ctx: 请求上下文。
	//  - tokenToRevoke: 需要吊销的令牌字符串（由调用方决定是吊销哪个令牌，通常建议吊销 Refresh Token）。
	// 返回:
	//  - error: 操作过程中发生的任何错误。注意，即使令牌解析失败或已过期，也可能视为“退出成功”，因为目标状态已达到。
	Logout(ctx context.Context, tokenToRevoke string) error

	// RefreshToken 使用有效的 Refresh Token 获取新的 Access Token 和 Refresh Token。
	// 主要逻辑: 解析传入的 Refresh Token，验证其有效性（签名、过期时间、是否在黑名单中），
	// 查询用户信息，生成新的令牌对，并将旧的 Refresh Token 加入黑名单。
	// 参数:
	//  - ctx: 请求上下文。
	//  - refreshToken: 用户持有的、用于请求续期的 Refresh Token 字符串。
	// 返回:
	//  - vo.TokenPair: 包含新的 Access Token 和 Refresh Token 的结构体。
	//  - error: 操作过程中发生的任何错误，可能是业务错误（如令牌无效、用户状态异常）或系统错误。
	RefreshToken(ctx context.Context, refreshToken string) (vo.TokenPair, error)
}

// authTokenService 是 AuthTokenService 接口的实现。
type authTokenService struct {
	tokenBlackRepo redis.TokenBlackRepo           // tokenBlackRepo: JTI 黑名单仓库。
	userRepo       mysql.UserRepository           // userRepo: 用户仓库，用于获取用户信息。
	jwtUtil        dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于解析和生成令牌。
	logger         *core.ZapLogger                // logger: 日志记录器。
}

// NewAuthTokenService 创建一个新的 authTokenService 实例。
// 设计原因:
// - 依赖注入确保了服务的可测试性和灵活性。
func NewAuthTokenService(
	tokenBlackRepo redis.TokenBlackRepo,
	userRepo mysql.UserRepository,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) AuthTokenService { // 返回接口类型
	return &authTokenService{ // 返回结构体指针
		tokenBlackRepo: tokenBlackRepo,
		userRepo:       userRepo,
		jwtUtil:        jwtUtil,
		logger:         logger, // 存储 logger
	}
}

// Logout 实现接口方法，处理退出登录。
func (s *authTokenService) Logout(ctx context.Context, tokenToRevoke string) error {
	const operation = "AuthTokenService.Logout"

	// 1. 解析需要吊销的令牌，获取 JTI 和过期时间
	//    这里假设优先尝试按 Refresh Token 解析，如果失败再尝试按 Access Token 解析
	//    或者调用方明确告知吊销的是哪种令牌。
	//    简化处理：我们假设调用方传入的是 Refresh Token。
	claims, err := s.jwtUtil.ParseRefreshToken(tokenToRevoke)
	if err != nil {
		// 尝试按 Access Token 解析 (如果需要同时吊销 Access Token 的 JTI)
		// claims, err = s.jwtUtil.ParseAccessToken(tokenToRevoke)
		// if err != nil {
		s.logger.Warn("退出登录时解析令牌失败或令牌无效",
			zap.String("operation", operation),
			// 不记录完整的 tokenToRevoke，可能过长或敏感
			zap.Error(err),
		)
		// 即使解析失败（例如令牌已过期或格式错误），从用户的角度看，“退出”的目标（令牌无法使用）
		// 已经达到或即将达到，所以可以认为操作成功，返回 nil。
		return nil
		// }
	}

	// 2. 计算令牌剩余的有效时间 (TTL)
	//    将 JTI 加入黑名单时，设置的过期时间应等于令牌本身的剩余有效时间。
	var ttl time.Duration
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time) // time.Until 自动处理已过期的情况 (返回负数或零)
	} else {
		// 如果令牌没有过期时间（不符合规范，但做防御性处理），可以设置一个默认的较短过期时间
		// 或者直接报错。这里我们选择记录警告并跳过黑名单（因为它没有明确的失效时间点）。
		s.logger.Warn("尝试吊销的令牌缺少过期时间声明",
			zap.String("operation", operation),
			zap.String("jti", claims.ID),
			zap.String("userID", claims.UserID),
		)
		return nil // 视为成功，因为无法确定黑名单时长
	}

	// 3. 将 JTI 加入黑名单
	//    只有当令牌尚未完全过期时，加入黑名单才有意义。
	if ttl > 0 {
		err = s.tokenBlackRepo.AddJtiToBlacklist(ctx, claims.ID, ttl)
		if err != nil {
			// 记录加入黑名单失败的错误，但通常不应阻塞用户退出流程
			s.logger.Error("将 JTI 加入黑名单失败",
				zap.String("operation", operation),
				zap.String("jti", claims.ID),
				zap.String("userID", claims.UserID),
				zap.Duration("ttl", ttl),
				zap.Error(err),
			)
			// 即使加入黑名单失败，也向上层返回成功，因为令牌很快会自然过期。
			// 如果对安全性要求极高，这里可以返回 commonerrors.ErrSystemError。
			return nil // 当前策略：不阻塞退出
		}
		s.logger.Info("成功将 JTI 加入黑名单",
			zap.String("operation", operation),
			zap.String("jti", claims.ID),
			zap.String("userID", claims.UserID),
			zap.Duration("ttl", ttl),
		)
	} else {
		s.logger.Info("令牌已过期，无需加入黑名单",
			zap.String("operation", operation),
			zap.String("jti", claims.ID),
			zap.String("userID", claims.UserID),
		)
	}

	// 4. 成功退出
	return nil
}

// RefreshToken 实现接口方法，刷新认证令牌。
func (s *authTokenService) RefreshToken(ctx context.Context, refreshToken string) (vo.TokenPair, error) {
	const operation = "AuthTokenService.RefreshToken"
	emptyTokenPair := vo.TokenPair{}

	// 1. 解析 Refresh Token 获取声明 (Claims)
	claims, err := s.jwtUtil.ParseRefreshToken(refreshToken)
	if err != nil {
		s.logger.Warn("解析 Refresh Token 失败或令牌无效",
			zap.String("operation", operation),
			zap.Error(err),
		)
		// 返回明确的业务错误
		return emptyTokenPair, errors.New("无效的刷新令牌")
	}
	// 从 Claims 中获取 JTI 和 UserID
	jti := claims.ID
	userID := claims.UserID

	// 2. 检查 Refresh Token 的 JTI 是否在黑名单中
	isBlacklisted, err := s.tokenBlackRepo.IsJtiBlacklisted(ctx, jti)
	if err != nil {
		// 检查黑名单时发生错误
		s.logger.Error("检查 JTI 黑名单失败",
			zap.String("operation", operation),
			zap.String("jti", jti),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyTokenPair, commonerrors.ErrSystemError
	}
	if isBlacklisted {
		// JTI 在黑名单中，表示此 Refresh Token 已被吊销（例如，用户已退出登录）
		s.logger.Warn("尝试使用已加入黑名单的 Refresh Token",
			zap.String("operation", operation),
			zap.String("jti", jti),
			zap.String("userID", userID),
		)
		return emptyTokenPair, errors.New("刷新令牌已失效") // 返回业务错误
	}

	// 3. 获取最新的用户信息
	//    需要用户信息来生成新的令牌，并检查用户状态。
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("刷新令牌时获取用户信息失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("jti", jti),
			zap.Error(err),
		)
		if errors.Is(err, commonerrors.ErrRepoNotFound) {
			// 用户不存在，数据异常
			return emptyTokenPair, errors.New("用户不存在，无法刷新令牌")
		}
		return emptyTokenPair, commonerrors.ErrSystemError
	}

	// 4. 检查用户状态是否允许刷新令牌
	if user.Status != enums.StatusActive {
		s.logger.Warn("尝试刷新令牌但用户状态异常",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.String("jti", jti),
			zap.Any("status", user.Status),
		)
		return emptyTokenPair, fmt.Errorf("用户状态异常，无法刷新令牌")
	}

	// 5. 生成新的 Access Token 和 Refresh Token
	//    平台信息从旧的 Refresh Token Claims 中获取，保持一致性
	platform := claims.Platform
	newAccessToken, err := s.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, platform)
	if err != nil {
		s.logger.Error("生成新的 Access Token 失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyTokenPair, commonerrors.ErrSystemError
	}
	newRefreshToken, err := s.jwtUtil.GenerateRefreshToken(user.UserID, platform)
	if err != nil {
		s.logger.Error("生成新的 Refresh Token 失败",
			zap.String("operation", operation),
			zap.String("userID", userID),
			zap.Error(err),
		)
		return emptyTokenPair, commonerrors.ErrSystemError
	}

	// 6. 将旧的 Refresh Token 加入黑名单
	//    计算旧 Refresh Token 的剩余 TTL
	var oldTokenTTL time.Duration
	if claims.ExpiresAt != nil {
		oldTokenTTL = time.Until(claims.ExpiresAt.Time)
	}
	// 只有当旧 Token 还有剩余时间时才加入黑名单
	if oldTokenTTL > 0 {
		err = s.tokenBlackRepo.AddJtiToBlacklist(ctx, jti, oldTokenTTL)
		if err != nil {
			// 加入黑名单失败是次要问题，记录日志但不应阻止令牌刷新成功返回
			s.logger.Error("将旧 Refresh Token JTI 加入黑名单失败",
				zap.String("operation", operation),
				zap.String("jti", jti),
				zap.String("userID", userID),
				zap.Duration("ttl", oldTokenTTL),
				zap.Error(err),
			)
			// 继续执行，返回新的令牌
		} else {
			s.logger.Info("成功将旧 Refresh Token JTI 加入黑名单",
				zap.String("operation", operation),
				zap.String("jti", jti),
				zap.String("userID", userID),
				zap.Duration("ttl", oldTokenTTL),
			)
		}
	}

	// 7. 成功刷新，返回新的令牌对
	s.logger.Info("成功刷新令牌",
		zap.String("operation", operation),
		zap.String("userID", userID),
		zap.String("oldJti", jti),
		// 不记录新令牌的具体内容
	)
	newTokenPair := vo.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}
	return newTokenPair, nil
}
