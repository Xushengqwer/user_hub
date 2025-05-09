package dependencies

import (
	"context"
	"encoding/json"
	"fmt" // 引入 fmt 包用于错误包装
	"github.com/Xushengqwer/user_hub/config"
	"io" // 引入 io 包读取响应体
	"net/http"
	"time"
)

// WechatClient 定义了与微信小程序服务端 API 交互的客户端接口。
// - 主要功能是根据小程序前端获取的 code 换取用户的 openid 和 session_key。
type WechatClient interface {
	// GetSession 使用小程序授权码换取 openid 和 session_key。
	// - ctx: 用于控制请求的上下文，例如超时或取消。
	// - code: 小程序通过 wx.login() 获取的临时登录凭证。
	// - 返回: openid (用户唯一标识), sessionKey (会话密钥), 以及可能的错误。
	// - 如果微信 API 返回错误码，会封装成 error 返回。
	GetSession(ctx context.Context, code string) (openid, sessionKey string, err error)
}

// wechatClient 是 WechatClient 接口的实现。
type wechatClient struct {
	config *config.WechatConfig // config 存储微信小程序的 AppID 和 Secret
	client *http.Client         // client 是用于发送 HTTP 请求的客户端实例
}

// wechatSessionResponse 定义了微信 jscode2session API 成功响应的结构。
// - 用于更清晰地解析 JSON 数据。
type wechatSessionResponse struct {
	OpenID     string `json:"openid"`      // 用户唯一标识
	SessionKey string `json:"session_key"` // 会话密钥
	UnionID    string `json:"unionid"`     // 用户在开放平台的唯一标识符，在满足UnionID下发条件时返回
	ErrCode    int    `json:"errcode"`     // 错误码 (成功时为 0)
	ErrMsg     string `json:"errmsg"`      // 错误信息 (成功时为 "ok")
}

// NewWechatClient 创建一个新的 wechatClient 实例。
// - 依赖注入微信配置和 HTTP 客户端。
func NewWechatClient(config *config.WechatConfig) WechatClient {
	return &wechatClient{
		config: config,
		client: &http.Client{
			// 设置合理的 HTTP 请求超时时间
			Timeout: 10 * time.Second,
		},
	}
}

// GetSession 实现接口方法，调用微信 API 获取会话信息。
func (w *wechatClient) GetSession(ctx context.Context, code string) (string, string, error) {
	// 1. 构造请求 URL
	// - 使用 fmt.Sprintf 安全地格式化 URL，包含 appid, secret 和 js_code。
	apiURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		w.config.AppID, w.config.Secret, code,
	)

	// 2. 创建 HTTP GET 请求
	// - 使用 http.NewRequestWithContext 传递上下文，允许超时和取消。
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		// 包装创建请求时的错误
		return "", "", fmt.Errorf("wechatClient.GetSession: 创建微信 API 请求失败: %w", err)
	}

	// 3. 发送 HTTP 请求
	resp, err := w.client.Do(req)
	if err != nil {
		// 包装发送请求时的错误 (例如网络问题、超时)
		return "", "", fmt.Errorf("wechatClient.GetSession: 请求微信 API 失败: %w", err)
	}
	// 确保响应体在使用后关闭，防止资源泄露
	defer resp.Body.Close()

	// 4. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 包装读取响应体时的错误
		return "", "", fmt.Errorf("wechatClient.GetSession: 读取微信 API 响应体失败: %w", err)
	}

	// 5. 检查 HTTP 状态码 (可选但推荐)
	// - 微信 API 通常在 body 中返回错误码，但也可能返回非 200 状态码
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("wechatClient.GetSession: 微信 API 返回非 200 状态码: %d, 响应体: %s", resp.StatusCode, string(body))
	}

	// 6. 解析 JSON 响应
	// - 使用具名结构体 wechatSessionResponse 解析响应体。
	var result wechatSessionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		// 包装 JSON 解析错误
		return "", "", fmt.Errorf("wechatClient.GetSession: 解析微信 API 响应失败: %w", err)
	}

	// 7. 检查微信业务错误码
	// - 如果 ErrCode 不为 0，表示微信 API 返回了业务错误。
	if result.ErrCode != 0 {
		// 返回包含微信错误码和错误信息的错误
		return "", "", fmt.Errorf("wechatClient.GetSession: 微信 API 业务错误: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	// 8. 成功获取，返回 openid 和 sessionKey
	return result.OpenID, result.SessionKey, nil
}
