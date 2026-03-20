package logger

import (
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

var (
	global   atomic.Value
	initOnce sync.Once
)

func New(env string) *zap.Logger {
	var log *zap.Logger
	if env == "production" {
		log, _ = zap.NewProduction()
	} else {
		log, _ = zap.NewDevelopment()
	}
	global.Store(log)
	return log
}

func get() *zap.Logger {
	if v := global.Load(); v != nil {
		return v.(*zap.Logger)
	}
	initOnce.Do(func() {
		l, _ := zap.NewDevelopment()
		global.Store(l)
	})
	return global.Load().(*zap.Logger)
}

func Info(msg string, fields ...zap.Field)  { get().Info(msg, fields...) }
func Error(msg string, fields ...zap.Field) { get().Error(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { get().Warn(msg, fields...) }
func Debug(msg string, fields ...zap.Field) { get().Debug(msg, fields...) }

func String(key, val string) zap.Field         { return zap.String(key, val) }
func Int(key string, val int) zap.Field         { return zap.Int(key, val) }
func Int64(key string, val int64) zap.Field     { return zap.Int64(key, val) }
func Bool(key string, val bool) zap.Field       { return zap.Bool(key, val) }
func Any(key string, val interface{}) zap.Field { return zap.Any(key, val) }
