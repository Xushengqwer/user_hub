package utils

import "golang.org/x/crypto/bcrypt"

// SetPassword 生成哈希密码（使用 bcrypt 哈希）
func SetPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword 校验用户输入的密码是否与存储的哈希密码匹配
func CheckPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err
}
