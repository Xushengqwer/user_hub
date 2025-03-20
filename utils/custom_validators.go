package utils

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"user_hub/models/enums"

	"regexp"
	"unicode"
)

var (
	phoneNumberRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// 预编译正则，提升性能
	usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]{1,20}$`)
)

// ValidateChinesePhone 只验证中国大陆手机号
func ValidateChinesePhone(fl validator.FieldLevel) bool {
	phoneNumber := fl.Field().String()
	return phoneNumberRegex.MatchString(phoneNumber)
}

// ValidateNickname 要求：只包含字母、数字和下划线，且长度≤20
func ValidateNickname(fl validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String())
}

// ValidatePassword 要求：6~30位，必须包含至少一个数字和一个字母
func ValidatePassword(fl validator.FieldLevel) bool {
	pwd := fl.Field().String()
	length := len(pwd)
	if length < 6 || length > 30 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, char := range pwd {
		if unicode.IsLetter(char) {
			hasLetter = true
		} else if unicode.IsDigit(char) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// 1.  先检查是否为空，如果为空，则不需要往下做验证了
// 2.  然后看能不能断言成我们的指针类型，如果可以再进行解引用，和我们实际允许的枚举值进行比较验证

func ValidGender(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.IsZero() || field.Interface() == nil {
		return true // 如果是 nil，跳过验证（假设 nil 是合法的）
	}
	val, ok := field.Interface().(*enums.Gender)
	if !ok {
		return false
	}
	return *val == enums.Female || *val == enums.Male || *val == enums.Unknown
}

func ValidStatus(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.IsZero() || field.Interface() == nil {
		return true
	}
	val, ok := field.Interface().(*enums.UserStatus)
	if !ok {
		return false
	}
	return *val == enums.Active || *val == enums.Blacklisted
}

func ValidRole(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.IsZero() || field.Interface() == nil {
		return true
	}
	val, ok := field.Interface().(*enums.UserRole)
	if !ok {
		return false
	}
	return *val == enums.Admin || *val == enums.User || *val == enums.Guest
}

// RegisterCustomValidators 注册自定义验证器
func RegisterCustomValidators() error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validations := map[string]validator.Func{
			"ChinesePhone": ValidateChinesePhone,
			"Username":     ValidateNickname,
			"Password":     ValidatePassword,
			"Status":       ValidStatus,
			"Role":         ValidRole,
			"Gender":       ValidGender,
		}

		for tag, validation := range validations {
			if err := v.RegisterValidation(tag, validation); err != nil {
				return fmt.Errorf("注册验证器 '%s' 失败: %w", tag, err)
			}
		}
	}
	return nil
}
