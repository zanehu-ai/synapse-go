package obs

import "context"

// 使用私有 struct 类型作为 context key，避免与其他包的字符串 key 碰撞（Go 最佳实践）。
type ctxKey struct{ name string }

var (
	requestIDKey   = ctxKey{"request_id"}
	tenantIDKey    = ctxKey{"tenant_id"}
	principalIDKey = ctxKey{"principal_id"}
)

// ── Request-ID ────────────────────────────────────────────────────────────────

// WithRequestID 将 request_id 注入 context，返回新 context。
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext 从 context 中读取 request_id。
// 若不存在，返回 ("", false)。
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// ── Tenant-ID ─────────────────────────────────────────────────────────────────

// WithTenantID 将 tenant_id 注入 context，返回新 context。
func WithTenantID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// TenantIDFromContext 从 context 中读取 tenant_id。
// 若不存在，返回 (0, false)。
func TenantIDFromContext(ctx context.Context) (uint64, bool) {
	v, ok := ctx.Value(tenantIDKey).(uint64)
	if !ok {
		return 0, false
	}
	return v, true
}

// ── Principal-ID ──────────────────────────────────────────────────────────────

// WithPrincipalID 将 principal_id 注入 context，返回新 context。
func WithPrincipalID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, principalIDKey, id)
}

// PrincipalIDFromContext 从 context 中读取 principal_id。
// 若不存在，返回 (0, false)。
func PrincipalIDFromContext(ctx context.Context) (uint64, bool) {
	v, ok := ctx.Value(principalIDKey).(uint64)
	if !ok {
		return 0, false
	}
	return v, true
}

// ── Outbox metadata helper ────────────────────────────────────────────────────

// RequestMetadataFromContext 从 context 提取请求级别元数据，返回 map[string]string。
// 可选择性地注入 outbox 事件 payload 的 metadata 字段，供故障定位时跨越 outbox 边界追踪。
//
// 使用方式（opt-in，不修改任何现有 outbox 调用方）：
//
//	evt := &outbox.Event{
//	    Type:     "payment.completed",
//	    Payload:  payloadBytes,
//	    Metadata: obs.RequestMetadataFromContext(ctx),
//	}
//
// 返回的 map 仅包含实际存在的字段（缺少则不写入 key），从不返回 nil。
func RequestMetadataFromContext(ctx context.Context) map[string]string {
	m := make(map[string]string, 3)
	if rid, ok := RequestIDFromContext(ctx); ok {
		m[FieldRequestID] = rid
	}
	if tid, ok := TenantIDFromContext(ctx); ok {
		m[FieldTenantID] = uint64ToString(tid)
	}
	if pid, ok := PrincipalIDFromContext(ctx); ok {
		m[FieldPrincipalID] = uint64ToString(pid)
	}
	return m
}

// uint64ToString 将 uint64 转换为十进制字符串，避免引入 strconv import 以外的依赖。
func uint64ToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	// 使用简单迭代，避免 fmt.Sprintf 的分配开销
	buf := [20]byte{}
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[pos:])
}
