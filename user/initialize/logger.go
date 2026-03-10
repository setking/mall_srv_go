package initialize

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() {
	// 日志目录
	logDir := "./logs"
	_ = os.MkdirAll(logDir, 0755)

	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// 文件 writer（带轮转）
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logDir + "/app.log",
		MaxSize:    100, // MB
		MaxBackups: 7,
		MaxAge:     30, // 天
		Compress:   true,
	})

	core := zapcore.NewTee(
		// 控制台：保持和原来一样的开发友好格式
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		),
		// 新增：写入本地文件（JSON格式，方便后续接入采集）
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			fileWriter,
			zapcore.DebugLevel,
		),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(logger)
}
