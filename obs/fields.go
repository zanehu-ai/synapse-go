package obs

// 标准字段名常量。所有服务层 slog 调用必须使用这些常量，禁止裸字符串字段名，
// 以保证全链路 grep 可用（例：grep tenant_id 能跨服务聚合）。
//
// 字段类型注释仅供参考；slog 使用 any 类型但会自动选择最合适的 Value 类型。
const (
	// FieldTenantID 租户 ID（uint64）。
	FieldTenantID = "tenant_id"

	// FieldPrincipalID 操作人 ID（uint64）。
	FieldPrincipalID = "principal_id"

	// FieldRequestID 请求唯一标识（UUID v4/v7 string），与 X-Request-ID HTTP 头对应。
	FieldRequestID = "request_id"

	// FieldEventType outbox 事件类型（string），如 "payment.completed"。
	FieldEventType = "event_type"

	// FieldErrorClass 类型化错误分类（string），如 "errs.ErrTenantMismatch"，
	// 便于按错误类型聚合告警，不依赖 error.Error() 消息内容。
	FieldErrorClass = "error_class"

	// FieldDurationMs 操作耗时（int64 毫秒）。
	FieldDurationMs = "duration_ms"

	// FieldOperation 服务方法名称（string），格式建议 "service.Method"，
	// 如 "PaymentService.CreateOrder"。
	FieldOperation = "operation"

	// FieldOutcome 操作结果（string）："success" | "failure" | "skipped"。
	FieldOutcome = "outcome"
)
