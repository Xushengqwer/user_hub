package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"user_hub/common/config"
)

// WechatClient 定义微信客户端接口
type WechatClient interface {
	// GetSession 使用小程序授权码换取 openid 和 session_key
	// - 输入: ctx 用于超时和取消控制，code 是小程序传递的临时授权码
	// - 输出: openid 是用户唯一标识，sessionKey 是会话密钥，err 表示可能的错误
	GetSession(ctx context.Context, code string) (openid, sessionKey string, err error)
}

// wechatClient 实现 WechatClient 接口
type wechatClient struct {
	config *config.WechatConfig
	client *http.Client
}

// NewWechatClient 创建微信客户端实例
// - 输入: config 包含 AppID 和 Secret
// - 输出: WechatClient 接口实例
func NewWechatClient(config *config.WechatConfig) WechatClient {
	return &wechatClient{
		config: config,
		client: &http.Client{
			Timeout: 10 * time.Second, // 设置默认超时时间
		},
	}
}

// GetSession 实现获取微信小程序会话信息
func (w *wechatClient) GetSession(ctx context.Context, code string) (string, string, error) {
	// 构造微信 jscode2session API 的请求 URL
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		w.config.AppID, w.config.Secret, code,
	)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建微信请求失败: %v", err)
	}

	// 发送请求
	resp, err := w.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求微信 API 失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"` // 可选，如果绑定了开放平台
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("解析微信响应失败: %v", err)
	}

	// 检查微信 API 返回的错误
	if result.ErrCode != 0 {
		return "", "", fmt.Errorf("微信 API 错误: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	// 返回 openid 和 session_key
	return result.OpenID, result.SessionKey, nil
}
