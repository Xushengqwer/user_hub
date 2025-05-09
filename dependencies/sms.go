package dependencies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Xushengqwer/user_hub/config"
	"net/http"
	"time"
)

//  todo 配置还没弄好，微信云托管SMS  API细节不确定是不是最新的

// SMSClient 定义短信验证码客户端接口
// - 用于发送验证码到用户手机号，支持第三方短信服务（如阿里云、腾讯云）
type SMSClient interface {
	// SendCode 发送验证码到指定手机号
	// - 输入: ctx 用于上下文控制，phone 是目标手机号，code 是生成的验证码
	// - 输出: error 表示发送是否成功，成功时返回 nil
	// - 注意: 不负责生成或存储验证码，仅处理发送逻辑
	SendCode(ctx context.Context, phone string, code string) error
}

// smsClient 实现 SMSClient 接口的结构体
type smsClient struct {
	config     *config.SMSConfig // 微信 SMS 服务配置
	httpClient *http.Client      // HTTP 客户端，用于发送请求
}

// NewSMSClient 创建 SMSClient 实例，通过依赖注入初始化
// - 输入: config 包含微信云托管 SMS 的配置信息
// - 输出: SMSClient 接口实例
// - 注意: 若配置为空，会导致初始化失败，需在调用前校验
func NewSMSClient(config *config.SMSConfig) (SMSClient, error) {
	// 1. 校验配置是否有效
	// - 确保必要字段非空
	if config == nil || config.AppID == "" || config.Secret == "" || config.Endpoint == "" || config.TemplateID == "" {
		fmt.Println(config)
		return nil, fmt.Errorf("SMS 配置无效，缺少必要字段")
	}

	// 2. 初始化 HTTP 客户端
	// - 设置默认超时为 10 秒
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 3. 返回 SMSClient 实例
	return &smsClient{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// SendCode 发送验证码到指定手机号
func (s *smsClient) SendCode(ctx context.Context, phone string, code string) error {
	// 1. 构造请求参数
	// - 根据微信云托管 SMS API 的要求，组装 JSON 数据
	// - 假设需要 AppID、Secret、手机号、模板 ID 和验证码
	reqBody := map[string]interface{}{
		"appid":       s.config.AppID,
		"secret":      s.config.Secret,
		"env":         s.config.Env,
		"template_id": s.config.TemplateID,
		"phone":       phone,
		"data": map[string]string{
			"code": code, // 模板中的验证码变量
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		// - JSON 序列化失败，返回错误
		return fmt.Errorf("构造短信请求参数失败: %v", err)
	}

	// 2. 创建 HTTP 请求
	// - 使用 POST 方法发送到微信云托管 SMS 端点
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.Endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		// - 创建请求失败，返回错误
		return fmt.Errorf("创建短信请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 3. 发送请求
	// - 使用 HTTP 客户端发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		// - 发送失败，返回错误
		return fmt.Errorf("发送短信验证码失败: %v", err)
	}
	defer resp.Body.Close()

	// 4. 检查响应状态
	// - 假设微信云托管返回 JSON，包含 errcode 和 errmsg
	// - errcode = 0 表示成功
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// - 解析响应失败，返回错误
		return fmt.Errorf("解析短信响应失败: %v", err)
	}

	// 5. 验证发送结果
	// - 检查 errcode 是否为 0
	if result.ErrCode != 0 {
		return fmt.Errorf("短信发送失败，错误码: %d, 错误信息: %s", result.ErrCode, result.ErrMsg)
	}

	// 6. 发送成功，返回 nil
	// - 表示验证码已成功发送到用户手机号
	return nil
}
