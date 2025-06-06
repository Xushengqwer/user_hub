package utils

import (
	"fmt"
	"github.com/Xushengqwer/go-common/models/enums"        // 导入公共模块的 enums 包
	myenums "github.com/Xushengqwer/user_hub/models/enums" // 导入项目内部的 enums 包，并使用别名 myenums 避免命名冲突
	"github.com/gin-gonic/gin/binding"                     // Gin 框架的数据绑定包
	"github.com/go-playground/validator/v10"               // 强大的数据校验库

	"regexp"  // 正则表达式包
	"unicode" // Unicode字符处理包
)

var (
	// phoneNumberRegex 预编译的中国大陆手机号正则表达式，用于提升校验性能。
	// 规则：以1开头，第二位是3到9之间的数字，后面跟9个数字。
	phoneNumberRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

	// usernameRegex 预编译的用户名（昵称）正则表达式，用于提升校验性能。
	// 规则：只包含大小写字母、数字和下划线，长度在1到20个字符之间。
	usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_]{1,20}$`)
)

// ValidateChinesePhone 校验是否为中国大陆手机号。
// fl: validator.FieldLevel 包含了当前校验字段的级别信息和值。
func ValidateChinesePhone(fl validator.FieldLevel) bool {
	phoneNumber := fl.Field().String()               // 获取字段的字符串表示
	return phoneNumberRegex.MatchString(phoneNumber) // 使用预编译的正则进行匹配
}

// ValidateNickname 校验昵称/用户账号格式。
// 要求：只包含字母、数字和下划线，且长度在1到20之间。
func ValidateNickname(fl validator.FieldLevel) bool {
	return usernameRegex.MatchString(fl.Field().String()) // 使用预编译的正则进行匹配
}

// ValidatePassword 校验密码格式。
// 要求：长度在6到30位之间，并且必须同时包含至少一个字母和一个数字。
func ValidatePassword(fl validator.FieldLevel) bool {
	pwd := fl.Field().String()
	length := len(pwd)
	if length < 6 || length > 30 { // 检查长度是否符合要求
		return false
	}
	var hasLetter, hasDigit bool // 标记是否包含字母和数字
	for _, char := range pwd {   // 遍历密码中的每个字符
		if unicode.IsLetter(char) { // 判断是否为字母
			hasLetter = true
		} else if unicode.IsDigit(char) { // 判断是否为数字
			hasDigit = true
		}
		if hasLetter && hasDigit { // 如果同时包含字母和数字，则校验通过
			return true
		}
	}
	return false // 如果遍历完仍未同时满足，则校验失败
}

// ValidGender 校验性别枚举值是否有效。
// 此校验器适用于指针类型的性别字段 (例如 *myenums.Gender)。
// 1. 先检查字段是否为零值 (例如 nil 指针) 或字段接口是否为 nil。如果是，则认为是有效的（通常表示该字段是可选的且未提供）。
// 2. 尝试将字段值断言为 *myenums.Gender 类型。如果断言失败，则无效。
// 3. 如果断言成功，解引用指针并检查其值是否为预定义的有效性别枚举值之一。
func ValidGender(fl validator.FieldLevel) bool {
	field := fl.Field() // 获取反射值
	// 如果字段是可选的 (omitempty)，且未提供值 (即为 nil)，则视为有效
	if field.IsZero() || field.Interface() == nil {
		return true
	}
	// 尝试类型断言为指向自定义 Gender 类型的指针
	val, ok := field.Interface().(*myenums.Gender)
	if !ok { // 类型不匹配，校验失败
		return false
	}
	// 检查解引用后的值是否为已定义的有效 Gender 枚举值
	return *val == myenums.Female || *val == myenums.Male || *val == myenums.Unknown
}

// ValidStatus 校验用户状态枚举值是否有效。
// 逻辑与 ValidGender 类似，但针对公共模块的 enums.UserStatus 类型。
func ValidStatus(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.IsZero() || field.Interface() == nil {
		return true
	}
	val, ok := field.Interface().(*enums.UserStatus) // 注意这里是公共模块的 UserStatus
	if !ok {
		return false
	}
	return *val == enums.StatusActive || *val == enums.StatusBlacklisted
}

// ValidRole 校验用户角色枚举值是否有效。
// 逻辑与 ValidGender 类似，但针对公共模块的 enums.UserRole 类型。
func ValidRole(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.IsZero() || field.Interface() == nil {
		return true
	}
	val, ok := field.Interface().(*enums.UserRole) // 注意这里是公共模块的 UserRole
	if !ok {
		return false
	}
	return *val == enums.RoleAdmin || *val == enums.RoleUser || *val == enums.RoleGuest
}

// RegisterCustomValidators 将所有自定义的校验函数注册到 Gin 的 validator 引擎中。
// 这样就可以在 DTO 的 struct tag 中使用这些自定义的校验标签了。
// 例如: `binding:"Account"` 或 `binding:"Password"`
func RegisterCustomValidators() error {
	// 获取 Gin 使用的 validator 实例
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 定义校验标签名和对应的校验函数
		validations := map[string]validator.Func{
			"ChinesePhone": ValidateChinesePhone, // 手机号校验
			"Account":      ValidateNickname,     // 账户名/昵称校验 (之前讨论中建议的标签名是 "Username"，这里是 "Account")
			"Password":     ValidatePassword,     // 密码格式校验
			"Status":       ValidStatus,          // 用户状态枚举校验
			"Role":         ValidRole,            // 用户角色枚举校验
			"Gender":       ValidGender,          // 性别枚举校验
		}

		// 遍历并注册所有自定义校验器
		for tag, validation := range validations {
			if err := v.RegisterValidation(tag, validation); err != nil {
				// 如果注册失败，返回错误信息，这通常会导致应用启动失败
				return fmt.Errorf("注册验证器 '%s' 失败: %w", tag, err)
			}
		}
	}
	return nil // 所有校验器注册成功
}
