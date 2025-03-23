package config

// SMSConfig 定义微信云托管 SMS 客户端的配置
type SMSConfig struct {
	// 微信云托管的 AppID
	AppID string `mapstructure:"appID" json:"appID" yaml:"appID"`

	// 微信云托管的 Secret
	Secret string `mapstructure:"secret" json:"secret" yaml:"secret"`

	// SMS 服务 API 端点（如 "https://api.weixin.qq.com/sms/send"）
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`

	// 短信模板 ID
	TemplateID string `mapstructure:"templateID" json:"templateID" yaml:"templateID"`

	// 云托管环境 ID（如 "prod-123"）
	Env string `mapstructure:"env" json:"env" yaml:"env"`
}
