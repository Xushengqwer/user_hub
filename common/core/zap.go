package core

import (
	"fmt"
	"os"
	"user_hub/common/config"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//  在应用关闭时，一般需要调用 logger.Sync() 以确保缓冲区日志被冲刷到文件！！！

// ZapLogger 封装了 *zap.Logger 实例，用于实际的日志记录操作
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger 是 ZapLogger 的构造函数，根据 ZapConfig 返回一个封装好的自定义的 *ZapLogger实例
func NewZapLogger(cfg config.ZapConfig) (*ZapLogger, error) {

	// 1. 解析日志级别

	// 1.1 定义一个 zapcore.level 类型的变量
	var level zapcore.Level
	// 1.2 使用zapcore.level的UnmarshalText 方法解析配置中的日志级别字符串
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("无效的日志级别: %v", err)
	}

	// 2. 定义日志的编码格式和字段映射，决定日志输出的结构和样式 （硬编码）

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",                         // 时间字段的键名，日志中将以 "time" 作为时间字段的名称。
		LevelKey:       "level",                        // 日志级别字段的键名，使用 "level"。
		NameKey:        "logger",                       // 日志记录器名称的键名，使用 "logger"。
		CallerKey:      "caller",                       // 调用者信息的键名，使用 "caller"。
		MessageKey:     "msg",                          // 日志消息的键名，使用 "msg"。
		StacktraceKey:  "stacktrace",                   //  堆栈跟踪信息的键名，使用 "stacktrace"。
		LineEnding:     zapcore.DefaultLineEnding,      //  行结束符，使用 Zap 的默认行结束符（通常是 \n）。
		EncodeLevel:    zapcore.CapitalLevelEncoder,    // 将日志级别字符串大写，例如 INFO，WARN
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // 定义时间的编码格式，ISO8601TimeEncoder 会将时间格式化为 ISO8601 标准格式（如 2023-04-05T14:30:00Z）。
		EncodeDuration: zapcore.SecondsDurationEncoder, // 定义持续时间的编码方式，SecondsDurationEncoder 会以秒为单位编码持续时间。
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 以短文件名+行号显示调用者
	}

	// 3. 选择编码器（JSON格式 或 Console控制台格式）

	var encoder zapcore.Encoder

	// 	   3.1 如果 cfg.Encoding 为 "json"，则使用 JSON 编码器 (NewJSONEncoder)。
	if cfg.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// 3.2 否则，使用控制台编码器 (NewConsoleEncoder)。
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 4. 设置普通日志（非错误日志）输出的目标
	regularWS, err := getWriteSyncer(cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	// 5. 设置错误日志输出的目标
	errorWS, err := getWriteSyncer(cfg.ErrorOutput)
	if err != nil {
		return nil, err
	}

	// 6. 定义日志级别过滤器  -- >>  根据日志级别过滤不同类型的日志，确保普通日志和错误日志被写入不同的输出目标。

	// 6.1 过滤普通日志（低于 ErrorLevel）。highPriority 用于过滤错误日志（ErrorLevel 及以上）。
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level && lvl < zapcore.ErrorLevel
	})
	// 6.2 过滤出 ErrorLevel 及以上的日志（如 error, fatal），用于错误日志输出。
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	// 7. 创建普通日志 core   -- >>    用于处理普通日志的编码和写入。
	regularCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(regularWS),
		lowPriority,
	)

	// 8. 创建错误日志 core   -- >>    用于处理错误日志的编码和写入。
	errorCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(errorWS),
		highPriority,
	)

	// 9. 合并 cores    -- >>  使用 zapcore.NewTee，它接受多个 Core，并将日志分发到所有符合条件的 Core。
	core := zapcore.NewTee(regularCore, errorCore)

	// 10.构建 Logger   -- >>       创建一个新的 Zap Logger 实例，配置其核心功能和选项。
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &ZapLogger{logger: logger}, nil
}

// 辅助函数：getWriteSyncer 根据 zapConfig 中的路径配置创建 WriteSyncer   -- >> 用于 Zap Core 的日志写入。
func getWriteSyncer(path string) (zapcore.WriteSyncer, error) {
	switch path {
	case "stdout":
		// 表示将日志输出到到标准输出
		return zapcore.AddSync(os.Stdout), nil
	case "stderr":
		// 表示将日志输出到标准错误。
		return zapcore.AddSync(os.Stderr), nil
	default:
		// 假设 path 是一个文件路径，使用 lumberjack.Logger 设置日志文件轮转策略：
		lumberjackLogger := &lumberjack.Logger{
			Filename:   path,
			MaxSize:    100,  // 单个日志文件的最大大小
			MaxBackups: 3,    // 保留的旧日志文件的最大数量
			MaxAge:     28,   // 保留旧日志文件的最大天数
			Compress:   true, // 是否压缩旧日志文件
		}
		return zapcore.AddSync(lumberjackLogger), nil
	}
}

// 为ZapLogger定义与zap.Logger一致的方法

func (z *ZapLogger) Debug(msg string, fields ...zap.Field) {
	z.logger.Debug(msg, fields...)
}

func (z *ZapLogger) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}

func (z *ZapLogger) Warn(msg string, fields ...zap.Field) {
	z.logger.Warn(msg, fields...)
}

func (z *ZapLogger) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}

func (z *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	z.logger.Fatal(msg, fields...)
}

// Logger 返回封装的 *zap.Logger 实例
func (z *ZapLogger) Logger() *zap.Logger {
	return z.logger
}
