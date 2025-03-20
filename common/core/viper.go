package core

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"user_hub/common/config"
)

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

	// 3. 配置文件路径（暂时使用绝对路径，因为我们启动的工作目录不确定）
	configFilePath := fmt.Sprintf("E:/doer_hub/common/config/config.%s.yaml", env)

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

	fmt.Println("配置初始化成功:", cfg)

	return &cfg, nil
}
