package initialize

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func OtelLoggerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		span := trace.SpanFromContext(ctx)
		spanCtx := span.SpanContext()

		fields := []zap.Field{
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
			zap.String("method", info.FullMethod),
		}

		start := time.Now()
		resp, err := handler(ctx, req)
		fields = append(fields, zap.Duration("latency", time.Since(start)))

		if err != nil {
			// ✅ 区分客户端错误(Warn) 和 服务端错误(Error)
			code := status.Code(err)
			if isClientError(code) {
				zap.L().Warn("gRPC请求失败", append(fields,
					zap.String("code", code.String()),
					zap.Error(err),
				)...)
			} else {
				zap.L().Error("gRPC请求失败", append(fields,
					zap.String("code", code.String()),
					zap.Error(err),
				)...)
			}
		} else {
			zap.L().Info("gRPC请求完成", fields...)
		}

		return resp, err
	}
}

// 客户端引起的错误，用 Warn 而不是 Error
func isClientError(code codes.Code) bool {
	switch code {
	case codes.InvalidArgument, // 参数错误
		codes.NotFound,         // 资源不存在
		codes.AlreadyExists,    // 重复创建
		codes.PermissionDenied, // 无权限
		codes.Unauthenticated:  // 未登录
		return true
	}
	return false
}
