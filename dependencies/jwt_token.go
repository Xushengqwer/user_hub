package dependencies

import (
	"errors"
	"fmt"
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	"github.com/google/uuid"
	"time"

	"github.com/golang-jwt/jwt/v5" // 引入 v5 版本的 JWT 包
)

// todo  当前使用 fmt.Sprintf("jti_%d", now.UnixNano()) 生成 JTI（JWT ID），但 UnixNano 在高并发场景下可能重复。
//	改进建议：使用 UUID 或其他唯一 ID 生成器（如 import "github.com/google/uuid"）
//	jti := uuid.New().String()

// JWTTokenInterface 定义 JWT 工具的接口
// - 用于生成和解析 JWT 令牌，提供访问令牌和刷新令牌的相关功能
type JWTTokenInterface interface {
	// GenerateAccessToken 生成访问令牌
	// - 输入: userID 用户ID, role 用户角色, status 用户状态, platform 客户端平台
	// - 输出: 访问令牌字符串和可能的错误
	GenerateAccessToken(userID string, role enums.UserRole, status enums.UserStatus, Platform enums.Platform) (string, error)

	// GenerateRefreshToken 生成刷新令牌
	// - 输入: userID 用户ID, platform 客户端平台
	// - 输出: 刷新令牌字符串和可能的错误
	GenerateRefreshToken(userID string, Platform enums.Platform) (string, error)

	// ParseAccessToken 解析并验证访问令牌
	// - 输入: tokenString 待解析的令牌字符串
	// - 输出: 解析后的 CustomClaims 和可能的错误
	ParseAccessToken(tokenString string) (*CustomClaims, error)

	// ParseRefreshToken 解析并验证刷新令牌
	// - 输入: tokenString 待解析的令牌字符串
	// - 输出: 解析后的 CustomClaims 和可能的错误
	ParseRefreshToken(tokenString string) (*CustomClaims, error)
}

// CustomClaims 定义 JWT 的声明结构体，包含标准字段和自定义字段
type CustomClaims struct {
	UserID               string           `json:"user_id"`  // 用户ID，唯一标识用户
	Role                 enums.UserRole   `json:"role"`     // 用户角色，例如管理员或普通用户
	Status               enums.UserStatus `json:"status"`   // 用户状态，例如活跃或禁用
	Platform             enums.Platform   `json:"platform"` // 客户端平台，例如 Web 或微信小程序
	jwt.RegisteredClaims                  // 嵌入 JWT v5 的标准声明字段
}

// JWTUtility 实现 JWTTokenInterface 接口的结构体
type JWTUtility struct {
	cfg *config.JWTConfig // JWT 配置，包含密钥、发行者等信息
}

// NewJWTUtility 创建 JWTUtility 实例，通过依赖注入初始化
// - 输入: cfg JWT 配置实例
// - 输出: JWTTokenInterface 接口实例
func NewJWTUtility(cfg *config.JWTConfig) JWTTokenInterface {
	return &JWTUtility{cfg: cfg}
}

// GenerateAccessToken 生成访问令牌
// - 输入: userID 用户ID, role 用户角色, status 用户状态, platform 客户端平台
// - 输出: 访问令牌字符串和可能的错误
func (ju *JWTUtility) GenerateAccessToken(userID string, role enums.UserRole, status enums.UserStatus, platform enums.Platform) (string, error) {
	now := time.Now()

	// 创建自定义声明
	claims := &CustomClaims{
		UserID:   userID,
		Role:     role,
		Status:   status,
		Platform: platform,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ju.cfg.Issuer,                                         // 令牌发行者，从配置中获取
			IssuedAt:  jwt.NewNumericDate(now),                               // 签发时间
			ExpiresAt: jwt.NewNumericDate(now.Add(constants.AccessTokenTTL)), // 过期时间，使用常量定义的 TTL
			ID:        uuid.New().String(),                                   // 默认生成唯一 JTI
		},
	}

	// 创建令牌，使用 HS256 签名算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用访问令牌的密钥签名
	secret := []byte(ju.cfg.SecretKey)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("签名令牌失败: %v", err)
	}
	return signedToken, nil
}

// GenerateRefreshToken 生成刷新令牌
// - 输入: userID 用户ID, platform 客户端平台
// - 输出: 刷新令牌字符串和可能的错误
func (ju *JWTUtility) GenerateRefreshToken(userID string, platform enums.Platform) (string, error) {
	now := time.Now()

	// 创建自定义声明
	claims := &CustomClaims{
		UserID:   userID,
		Platform: platform,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ju.cfg.Issuer,                                          // 令牌发行者，从配置中获取
			IssuedAt:  jwt.NewNumericDate(now),                                // 签发时间
			ExpiresAt: jwt.NewNumericDate(now.Add(constants.RefreshTokenTTL)), // 过期时间，使用常量定义的 TTL
			ID:        uuid.New().String(),                                    // 默认生成唯一 JTI
		},
	}

	// 创建令牌，使用 HS256 签名算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用刷新令牌的密钥签名
	secret := []byte(ju.cfg.RefreshSecret)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("签名令牌失败: %v", err)
	}
	return signedToken, nil
}

// ParseAccessToken 解析并验证访问令牌
// - 输入: tokenString 待解析的令牌字符串
// - 输出: 解析后的 CustomClaims 和可能的错误
func (ju *JWTUtility) ParseAccessToken(tokenString string) (*CustomClaims, error) {
	secret := []byte(ju.cfg.SecretKey)

	// 创建解析器，启用 v5 的严格验证选项
	parser := jwt.NewParser(
		jwt.WithExpirationRequired(),  // 强制要求令牌包含过期时间
		jwt.WithIssuer(ju.cfg.Issuer), // 验证发行者是否匹配配置中的值
	)

	// 解析令牌
	return ju.parseToken(tokenString, secret, parser)
}

// ParseRefreshToken 解析并验证刷新令牌
// - 输入: tokenString 待解析的令牌字符串
// - 输出: 解析后的 CustomClaims 和可能的错误
func (ju *JWTUtility) ParseRefreshToken(tokenString string) (*CustomClaims, error) {
	secret := []byte(ju.cfg.RefreshSecret)

	// 创建解析器，启用 v5 的严格验证选项
	parser := jwt.NewParser(
		jwt.WithExpirationRequired(),  // 强制要求令牌包含过期时间
		jwt.WithIssuer(ju.cfg.Issuer), // 验证发行者是否匹配配置中的值
	)

	// 解析令牌
	return ju.parseToken(tokenString, secret, parser)
}

// parseToken 辅助函数，用于解析和验证 JWT 令牌
// - 输入: tokenString 待解析的令牌字符串, secret 签名密钥, parser v5 的解析器实例
// - 输出: 解析后的 CustomClaims 和可能的错误
func (ju *JWTUtility) parseToken(tokenString string, secret []byte, parser *jwt.Parser) (*CustomClaims, error) {
	// 使用 v5 的 Parser 解析令牌
	token, err := parser.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法是否为 HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法不匹配: %v", token.Header["alg"])
		}
		return secret, nil
	})

	// 如果解析失败，返回错误
	if err != nil {
		return nil, err
	}

	// 类型断言并验证令牌有效性
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的JWT声明")
	}

	return claims, nil
}
