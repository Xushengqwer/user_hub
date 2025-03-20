package config

type GlobalConfig struct {
	JWTConfig       JWTConfig       `mapstructure:"jwtConfig" json:"jwtConfig" yaml:"jwtConfig"`
	MySQLConfig     MySQLConfig     `mapstructure:"mySQLConfig" json:"mySQLConfig" yaml:"mySQLConfig"`
	RedisConfig     RedisConfig     `mapstructure:"redisConfig" json:"redisConfig" yaml:"redisConfig"`
	ZapConfig       ZapConfig       `mapstructure:"zapConfig" json:"zapConfig" yaml:"zapConfig"`
	RateLimitConfig RateLimitConfig `mapstructure:"rateLimitConfig" json:"rateLimitConfig" yaml:"rateLimitConfig"`
	Server          Server          `mapstructure:"server" json:"server" yaml:"server"`
}
