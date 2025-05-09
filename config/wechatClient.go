package config

type WechatConfig struct {
	// 小程序的 AppID
	AppID string `mapstructure:"appID" json:"appID" yaml:"appID"`

	// 小程序的 AppSecret
	Secret string `mapstructure:"secret" json:"secret" yaml:"secret"`
}
