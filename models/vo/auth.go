package vo

type Userinfo struct {
	UserID string `json:"userID"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`  // 新认证令牌
	RefreshToken string `json:"refresh_token"` // 新刷新令牌（可选）
}

type LoginResponse struct {
	User  Userinfo  `json:"userManage"` // 用户信息
	Token TokenPair `json:"token"`      // Token 对
}
