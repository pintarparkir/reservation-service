package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger initialises the package-level zap logger.
// Production env writes to JSON + rotates via lumberjack; local writes to console.
func NewLogger(appName, appEnv string) error {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if appEnv == "local" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	rotator := &lumberjack.Logger{
		Filename:   "/var/log/" + appName + "/app.log",
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     30,
		Compress:   true,
	}
	writer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(rotator), zapcore.AddSync(consoleStdout()))

	core := zapcore.NewCore(encoder, writer, zapcore.InfoLevel)
	instance = zap.New(core, zap.AddCaller(), zap.Fields(zap.String("app", appName), zap.String("env", appEnv)))
	return nil
}
