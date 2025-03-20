package core

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// 确保 GormLogger 实现 gorm/logger.Interface
var _ logger.Interface = (*GormLogger)(nil)

// GormLogger 实现了 gorm/logger.Interface 接口，使用 ZapLogger
type GormLogger struct {
	zapLogger *ZapLogger
	logLevel  logger.LogLevel
}

// NewGormLogger 创建一个新的 GormLogger 实例
func NewGormLogger(zapLogger *ZapLogger) *GormLogger {
	return &GormLogger{
		zapLogger: zapLogger,
		logLevel:  logger.Info,
	}
}

// LogMode 设置日志级别
func (g *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	g.logLevel = level
	return g
}

func (g *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if g.logLevel >= logger.Info {
		g.zapLogger.Info(msg, zap.Any("args", args))
	}
}

func (g *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if g.logLevel >= logger.Warn {
		g.zapLogger.Warn(msg, zap.Any("args", args))
	}
}

func (g *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if g.logLevel >= logger.Error {
		g.zapLogger.Error(msg, zap.Any("args", args))
	}
}

func (g *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	if err != nil && g.logLevel >= logger.Error {
		g.zapLogger.Error("SQL 执行出错",
			zap.Error(err),
			zap.Duration("耗时", elapsed),
			zap.Int64("行数", rows),
			zap.String("SQL", sql),
		)
	} else if g.logLevel >= logger.Info {
		g.zapLogger.Info("SQL 执行",
			zap.Duration("耗时", elapsed),
			zap.Int64("行数", rows),
			zap.String("SQL", sql),
		)
	}
}
