package core

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strconv"
	"user_hub/common/config"
)

// todo  目前这里的的代码不优雅，有问题

// 示例   go run main.go --env=production  通过命令行参数运行，加载配置文件：./config/config.production.yaml
// 示例   ENV=development go run main.go   通过环境变量运行：加载配置文件：./config/config.development.yaml
// 示例   go run main.go       默认运行： 如果不提供环境变量或命令行参数，默认为 development：加载配置文件：./config/config.development.yaml// ViperLoadConfig 加载配置并返回 GlobalConfig 实例

func ViperLoadConfig() (*config.GlobalConfig, error) {
	v := viper.New()

	// 定义环境变量优先级
	var env string
	// 1. 先从环境变量中获取运行环境
	if os.Getenv("ENV") != "" {
		env = os.Getenv("ENV")
	} else {
		// 2. 如果没有环境变量，则从命令行参数中获取运行环境

		// 2.1 定义一个命令行参数，并设置默认值为——development
		flag.StringVar(&env, "env", "development", "设置运行环境，例如 development, production")
		// 2.2 解析命令行参数，绑定到 env 变量中
		flag.Parse()
	}

	// 3. 配置文件路径（使用相对路径加载配置文件）
	configFilePath := fmt.Sprintf("common/config/config.%s.yaml", env)

	// 4. 给viper设置好——配置文件的路径和类型
	v.SetConfigFile(configFilePath)
	v.SetConfigType("yaml")

	// 5. 读取配置文件到viper实例中，并处理错误
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 出错: %s \n", configFilePath, err)
	}
	fmt.Printf("成功加载配置文件：%s\n", configFilePath)

	// 6. 将读取到的配置解析到结构体中
	var cfg config.GlobalConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置到结构体出错: %s \n", err)
	}

	// 覆盖 MySQL 配置
	if host := os.Getenv("MY_SQL_HOST"); host != "" {
		cfg.MySQLConfig.Host = host
	}
	if port := os.Getenv("MY_SQL_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.MySQLConfig.Port = p
		}
	}
	if user := os.Getenv("MY_SQL_USER"); user != "" {
		cfg.MySQLConfig.User = user
	}
	if password := os.Getenv("MY_SQL_PASSWORD"); password != "" {
		cfg.MySQLConfig.Password = password
	}

	// 覆盖 Redis 配置
	if address := os.Getenv("REDIS_ADDRESS"); address != "" {
		cfg.RedisConfig.Address = address
	}
	if port := os.Getenv("REDIS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.RedisConfig.Port = p
		}
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		cfg.RedisConfig.Password = password
	}

	fmt.Println("配置初始化成功:", cfg)

	return &cfg, nil
}
