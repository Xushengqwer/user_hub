package dto

type AccountRegisterData struct {
	Account         string `json:"account" binding:"required"`         // 用户账号
	Password        string `json:"password" binding:"required"`        // 密码
	ConfirmPassword string `json:"confirmPassword" binding:"required"` // 密码
}

type AccountLoginData struct {
	Account  string `json:"account" binding:"required"`  // 用户账号
	Password string `json:"password" binding:"required"` // 密码
}
