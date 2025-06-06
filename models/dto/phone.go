package dto

// PhoneLoginOrRegisterData 定义手机号登录或注册的数据传输对象
type PhoneLoginOrRegisterData struct {
	Phone string `json:"phone" binding:"required"` // 手机号，必填
	Code  string `json:"code" binding:"required"`  // 验证码，必填
}

//	todo mobile标签还没实现呢

// SendCaptchaRequest 定义发送验证码的请求数据传输对象
type SendCaptchaRequest struct {
	Phone string `json:"phone" binding:"required,mobile"` // 手机号，必填且需符合格式
}
