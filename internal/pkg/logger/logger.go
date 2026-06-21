// Package logger 日志初始化
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 初始化 zap 日志
// mode: "debug" 输出到控制台 + 开发格式, "release" 输出 JSON 格式
func New(mode string) *zap.SugaredLogger {
	var cfg zap.Config

	if mode == "release" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	logger, err := cfg.Build()
	if err != nil {
		// 降级：无法初始化 zap 就用最基本输出
		os.Stderr.WriteString("初始化日志失败: " + err.Error() + "\n")
		os.Exit(1)
	}

	return logger.Sugar()
}
