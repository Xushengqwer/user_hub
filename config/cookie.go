package config

// CookieConfig 定义用于设置 HTTP Cookie 的相关参数
type CookieConfig struct {
	// Domain 指定 Cookie 对哪些域名有效。
	// 例如: "example.com" (对 example.com 生效，但不包括 www.example.com)
	//      ".example.com" (对 example.com 及其所有子域名生效)
	//      留空则表示仅对当前请求的主机生效（不包括子域名）。
	Domain string `mapstructure:"domain" json:"domain" yaml:"domain"`

	// Path 指定 Cookie 在哪些路径下有效。
	// 通常设为 "/" 使其对整个域名下的所有路径都有效。
	Path string `mapstructure:"path" json:"path" yaml:"path"`

	// Secure 标记指示浏览器仅通过 HTTPS 连接发送 Cookie。
	// 对于生产环境和任何涉及敏感信息的 Cookie (如刷新令牌)，应始终设为 true。
	Secure bool `mapstructure:"secure" json:"secure" yaml:"secure"`

	// HttpOnly 标记指示浏览器不允许通过 JavaScript (如 document.cookie) 访问 Cookie。
	// 这是防止 XSS 攻击窃取 Cookie 的关键安全措施，对于存储刷新令牌的 Cookie 必须设为 true。
	HttpOnly bool `mapstructure:"http_only" json:"http_only" yaml:"http_only"`

	// SameSite 控制 Cookie 是否随跨站请求发送，有助于缓解 CSRF 攻击。
	// 可选值: "Lax" (默认), "Strict", "None"。
	// - "Lax": 大多数情况下是好的默认值。Cookie 会在顶级导航GET请求时跨站发送。
	// - "Strict": Cookie 仅在完全相同的站点发起的请求中发送。
	// - "None": Cookie 会在所有跨站请求中发送，但必须同时设置 Secure=true。
	SameSite string `mapstructure:"same_site" json:"same_site" yaml:"same_site"`

	// RefreshTokenName 定义了存储刷新令牌的 Cookie 的名称。
	RefreshTokenName string `mapstructure:"refresh_token_name" json:"refresh_token_name" yaml:"refresh_token_name"`

	// 注意: 刷新令牌 Cookie 的 MaxAge (生命周期) 将从 constants.RefreshTokenTTL 获取并转换为秒。
}
