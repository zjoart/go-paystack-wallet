package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// project specific keys
const (
	RequestIDKey = "request_id"
	UserIdKey    = "user_id"
	ServiceKey   = "service"
	EnvKey       = "env"
	ErrorKey     = "error"
)

func init() {
	var err error
	config := zap.NewProductionConfig()

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.StacktraceKey = ""
	encoderConfig.CallerKey = "caller"
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"

	config.EncoderConfig = encoderConfig
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	Log, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

type Fields map[string]interface{}

func Info(msg string, fields ...Fields) {
	if len(fields) > 0 {
		Log.Info(msg, getZapFields(fields[0])...)
		return
	}
	Log.Info(msg)
}

func Error(msg string, fields ...Fields) {
	if len(fields) > 0 {
		Log.Error(msg, getZapFields(fields[0])...)
		return
	}
	Log.Error(msg)
}

func Debug(msg string, fields ...Fields) {
	if len(fields) > 0 {
		Log.Debug(msg, getZapFields(fields[0])...)
		return
	}
	Log.Debug(msg)
}

func Warn(msg string, fields ...Fields) {
	if len(fields) > 0 {
		Log.Warn(msg, getZapFields(fields[0])...)
		return
	}
	Log.Warn(msg)
}

func Fatal(msg string, fields ...Fields) {
	if len(fields) > 0 {
		Log.Fatal(msg, getZapFields(fields[0])...)
		return
	}
	Log.Fatal(msg)
}

// WithError adds an error field to the log entry
func WithError(err error) Fields {
	return Fields{
		ErrorKey: err.Error(),
	}
}

func Merge(fields ...Fields) Fields {
	merged := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}

func getZapFields(fields Fields) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return zapFields
}
