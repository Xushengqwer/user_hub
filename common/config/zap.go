package config

// ZapConfig 定义Zap日志框架的相关配置, 包含日志级别、日志文件路径、编码格式等信息

type ZapConfig struct {
	Level       string `mapstructure:"level" yaml:"level"`               // 日志级别，例如"debug", "info", "warn", "error"
	Encoding    string `mapstructure:"encoding" yaml:"encoding"`         // 编码格式，例如"json"或"console"
	OutputPath  string `mapstructure:"output_path" yaml:"output_path"`   // 普通日志输出路径，例如"stdout"或"./logs/app.log"
	ErrorOutput string `mapstructure:"error_output" yaml:"error_output"` // 错误日志输出路径，例如"stderr"或"./logs/error.log"
	MaxSize     int    `mapstructure:"max_size" yaml:"max_size"`         // 每个日志文件的最大大小，单位MB，超过会切割
	MaxAge      int    `mapstructure:"max_age" yaml:"max_age"`           // 日志文件保留的最大天数
	MaxBackups  int    `mapstructure:"max_backups" yaml:"max_backups"`   // 最多保留多少个备份日志文件
}
