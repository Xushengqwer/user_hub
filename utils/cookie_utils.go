package utils

import (
	"net/http"
	"strings"
)

// ParseSameSiteString 将字符串形式的 SameSite 配置转换为 http.SameSite 类型
func ParseSameSiteString(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		// 根据你的默认策略选择，Lax 通常是个不错的选择
		// 或者你可以返回一个错误或panic如果配置了无效的值
		return http.SameSiteLaxMode // 或者 http.SameSiteDefaultMode
	}
}
