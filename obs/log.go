package obs

import (
	"context"
	"log/slog"
)

// Logger is a platform structured logging wrapper around slog.Logger:
//   - WithRequest 自动从 context 读取 request_id / tenant_id / principal_id 并附加到日志
//   - Info / Warn / Error 接受 slog.Attr 或 key-value 交替参数（与 slog 惯用法一致）
//   - 幂等性：多次 WithRequest 不累积重复字段
//
// Logger 值本身是不可变的（每次 With* 返回新实例），可安全跨 goroutine 共享。
type Logger struct {
	inner *slog.Logger
}

// NewLogger 用给定的 slog.Handler 构造平台 Logger。
// 典型用法（cmd/api/main.go）：
//
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
//	logger := obs.NewLogger(handler)
func NewLogger(handler slog.Handler) *Logger {
	if handler == nil {
		handler = slog.Default().Handler()
	}
	return &Logger{inner: slog.New(handler)}
}

// Default 返回包装了 slog.Default() 的平台 Logger，便于单元测试及一次性工具使用。
func Default() *Logger {
	return &Logger{inner: slog.Default()}
}

// WithRequest 返回一个子 Logger，预附加 context 中存在的以下字段（缺少则跳过）：
//   - request_id
//   - tenant_id
//   - principal_id
//
// 调用方可在请求入口处（handler / middleware）调用一次，后续直接传子 Logger。
//
// 若 ctx 为 nil，等价于传入 context.Background()——不 panic，不附加任何字段。
func (l *Logger) WithRequest(ctx context.Context) *Logger {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := make([]any, 0, 6)
	if rid, ok := RequestIDFromContext(ctx); ok {
		attrs = append(attrs, slog.String(FieldRequestID, rid))
	}
	if tid, ok := TenantIDFromContext(ctx); ok {
		attrs = append(attrs, slog.Uint64(FieldTenantID, tid))
	}
	if pid, ok := PrincipalIDFromContext(ctx); ok {
		attrs = append(attrs, slog.Uint64(FieldPrincipalID, pid))
	}
	if len(attrs) == 0 {
		return l
	}
	return &Logger{inner: l.inner.With(attrs...)}
}

// With 返回添加了给定属性的子 Logger（直接透传 slog.Logger.With）。
func (l *Logger) With(attrs ...any) *Logger {
	return &Logger{inner: l.inner.With(attrs...)}
}

// Info 记录 INFO 级别日志。attrs 可以是 slog.Attr 或 key-value 交替对。
// ctx 用于从调用栈传递 slog source（不从 ctx 再次提取请求字段，由 WithRequest 提前注入）。
func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	l.inner.InfoContext(ctx, msg, attrs...)
}

// Warn 记录 WARN 级别日志。
func (l *Logger) Warn(ctx context.Context, msg string, attrs ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	l.inner.WarnContext(ctx, msg, attrs...)
}

// Error 记录 ERROR 级别日志。err 允许为 nil（不 panic）。
func (l *Logger) Error(ctx context.Context, msg string, err error, attrs ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	combined := make([]any, 0, len(attrs)+2)
	if err != nil {
		combined = append(combined, slog.Any("error", err))
	}
	combined = append(combined, attrs...)
	l.inner.ErrorContext(ctx, msg, combined...)
}

// Debug 记录 DEBUG 级别日志（生产默认关闭，测试/本地开发用）。
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	l.inner.DebugContext(ctx, msg, attrs...)
}

// Slog 返回底层 *slog.Logger，供需要直接访问 slog API 的场景（慎用）。
func (l *Logger) Slog() *slog.Logger {
	return l.inner
}
