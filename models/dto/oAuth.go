package dto

// WechatMiniProgramLoginData 定义 DTO 结构体，用于接收小程序授权数据
type WechatMiniProgramLoginData struct {
	// Code 微信小程序通过 wx.login() 获取的临时授权码
	// - 必填，用于后端换取 openid 和 session_key
	Code string `json:"code" binding:"required"`
}
