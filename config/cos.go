package config

// COSConfig 定义腾讯云对象存储 (COS) 的相关配置
type COSConfig struct {
	SecretID   string `mapstructure:"secret_id" yaml:"secret_id"`     // COS 的 SecretId
	SecretKey  string `mapstructure:"secret_key" yaml:"secret_key"`   // COS 的 SecretKey
	BucketName string `mapstructure:"bucket_name" yaml:"bucket_name"` // 存储桶名称（例如 doer-user-hub）
	AppID      string `mapstructure:"app_id" yaml:"app_id"`           // 存储桶的 APPID (数字部分)
	Region     string `mapstructure:"region" yaml:"region"`           // 存储桶所属地域 (例如 ap-guangzhou)
	BaseURL    string `mapstructure:"base_url" yaml:"base_url"`       // 可选：存储桶的访问基础 URL (例如 https://images.example.com)
}
