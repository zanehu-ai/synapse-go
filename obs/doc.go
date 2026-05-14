// Package obs provides platform-level structured observability helpers.
//
// 当前版本（Phase 4）覆盖两项最高优先级能力：
//
//  1. Structured logging wrapper (Logger), built on log/slog and standard
//     field names. This avoids field drift such as tenant_id / tenantID / tid.
//
//  2. Request-ID propagation through context and outbound HTTP calls.
//
// # Quick Start
//
//	// 1. Initialize a Logger in cmd/api/main.go.
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
//	logger := obs.NewLogger(handler)
//
//	// 2. Use it in handlers and services.
//	logger.Info(ctx, "payment processed",
//	    slog.Int64(obs.FieldDurationMs, latency.Milliseconds()),
//	    slog.String(obs.FieldOutcome, "success"),
//	)
//
//	// 3. Propagate request_id to outbound HTTP calls.
//	resp, err := obs.DoWithContext(ctx, httpClient, req)
//
//	// 4. Build request metadata for outbox/event payloads.
//	meta := obs.RequestMetadataFromContext(ctx)
//
// # Field Names
//
// Standard field names are exported as [Field*] constants in fields.go.
// Services should use these constants instead of raw strings.
//
// # Out Of Scope
//
//   - Prometheus metrics
//   - OpenTelemetry tracing
//   - Log sampling or dynamic log levels
package obs
