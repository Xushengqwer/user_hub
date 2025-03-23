package enums

import "errors"

// Platform 平台
type Platform string

const (
	Web    Platform = "web"    // 网站
	Wechat Platform = "wechat" // 微信小程序
	APP    Platform = "app"    // APP
)

// PlatformFromString 将字符串转换为 Platform 类型
// - 输入: s string，待转换的字符串
// - 输出: Platform 转换后的平台类型
// - 输出: error 如果字符串不是有效的平台类型，则返回错误
// - 意图: 确保传入的平台字符串是预定义的有效值，避免无效平台类型导致的错误
func PlatformFromString(s string) (Platform, error) {
	switch Platform(s) {
	case Web, Wechat, APP:
		return Platform(s), nil
	default:
		return "", errors.New("无效的平台类型")
	}
}
