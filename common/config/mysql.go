package config

// MySQLConfig 定义MySQL连接的相关配置
type MySQLConfig struct {
	Host        string `mapstructure:"host" yaml:"host"`                   // MySQL服务器地址
	Port        int    `mapstructure:"port" yaml:"port"`                   // MySQL服务器端口
	User        string `mapstructure:"user" yaml:"user"`                   // 数据库用户名
	Password    string `mapstructure:"password" yaml:"password"`           // 数据库密码
	Database    string `mapstructure:"database" yaml:"database"`           // 使用的数据库名称
	Charset     string `mapstructure:"charset" yaml:"charset"`             // 字符集，例如utf8mb4
	ParseTime   bool   `mapstructure:"parseTime" yaml:"parseTime"`         // 是否解析时间类型
	Loc         string `mapstructure:"loc" yaml:"loc"`                     // 时区
	MaxOpenConn int    `mapstructure:"max_open_conn" yaml:"max_open_conn"` // 最大打开连接数
	MaxIdleConn int    `mapstructure:"max_idle_conn" yaml:"max_idle_conn"` // 最大空闲连接数
}
