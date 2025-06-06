package config

// MySQLConfig 定义MySQL连接的相关配置
type MySQLConfig struct {
	DSN         string `mapstructure:"dsn" yaml:"dsn"`                     // MySQL DSN (Data Source Name)，例如 "userManage:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local"
	MaxOpenConn int    `mapstructure:"max_open_conn" yaml:"max_open_conn"` // 最大打开连接数
	MaxIdleConn int    `mapstructure:"max_idle_conn" yaml:"max_idle_conn"` // 最大空闲连接数
}
