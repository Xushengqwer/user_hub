package utils

import (
	"fmt"
	"math/rand/v2"
)

// GenerateCaptcha 生成6位随机验证码
func GenerateCaptcha() string {
	code := rand.IntN(900000) + 100000 // 生成 100000~999999 的随机数
	return fmt.Sprintf("%06d", code)   // 格式化为 6 位字符串，保证前导零
}
