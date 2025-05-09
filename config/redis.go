package config

import (
	"time"
)

// RedisConfig 定义Redis连接的相关配置
// 包含地址、端口、密码、数据库编号、连接超时等信息

// RedisConfig 定义Redis连接的相关配置
type RedisConfig struct {
	Address      string        `mapstructure:"address" yaml:"address"`               // Redis服务器地址
	Port         int           `mapstructure:"port" yaml:"port"`                     // Redis服务器端口
	Password     string        `mapstructure:"password" yaml:"password"`             // Redis密码
	DB           int           `mapstructure:"db" yaml:"db"`                         // 使用的Redis数据库编号
	DialTimeout  time.Duration `mapstructure:"dial_timeout" yaml:"dial_timeout"`     // 连接超时时间
	ReadTimeout  time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`     // 读取超时时间
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`   // 写入超时时间
	PoolSize     int           `mapstructure:"pool_size" yaml:"pool_size"`           // 连接池大小
	MinIdleConns int           `mapstructure:"min_idle_conns" yaml:"min_idle_conns"` // 最小空闲连接数
}
