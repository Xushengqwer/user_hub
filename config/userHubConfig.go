package config

import (
	"github.com/Xushengqwer/go-common/config"
)

type UserHubConfig struct {
	ZapConfig     config.ZapConfig     `mapstructure:"zapConfig" json:"zapConfig" yaml:"zapConfig"`
	GormLogConfig config.GormLogConfig `mapstructure:"gormLogConfig" json:"gormLogConfig" yaml:"gormLogConfig"`
	ServerConfig  config.ServerConfig  `mapstructure:"serverConfig" json:"serverConfig" yaml:"serverConfig"`
	TracerConfig  config.TracerConfig  `mapstructure:"tracerConfig" json:"tracerConfig" yaml:"tracerConfig"`
	JWTConfig     JWTConfig            `mapstructure:"jwtConfig" json:"jwtConfig" yaml:"jwtConfig"`
	MySQLConfig   MySQLConfig          `mapstructure:"mySQLConfig" json:"mySQLConfig" yaml:"mySQLConfig"`
	RedisConfig   RedisConfig          `mapstructure:"redisConfig" json:"redisConfig" yaml:"redisConfig"`
	WechatConfig  WechatConfig         `mapstructure:"wechatConfig" json:"wechatConfig" yaml:"wechatConfig"`
	SMSConfig     SMSConfig            `mapstructure:"smsConfig" json:"smsConfig" yaml:"smsConfig"`
	COSConfig     COSConfig            `mapstructure:"cosConfig" json:"cosConfig" yaml:"cosConfig"`
	CookieConfig  CookieConfig         `mapstructure:"cookieConfig" json:"cookieConfig" yaml:"cookieConfig"`
}
