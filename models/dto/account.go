package dto

type AccountRegisterData struct {
	Account         string `json:"account" binding:"required,Account"`   // 使用 "Account" 校验器
	Password        string `json:"password" binding:"required,Password"` // 使用 "Password" 校验器
	ConfirmPassword string `json:"confirmPassword" binding:"required"`   // 这里没有自定义格式校验器，但如果需要在服务端检查密码一致性，可以添加 `eqfield=Password`，不过这通常在前端或服务层处理。
}

type AccountLoginData struct {
	Account  string `json:"account" binding:"required"`  // 用户账号
	Password string `json:"password" binding:"required"` // 密码
}
