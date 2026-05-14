package obs

import (
	"context"
	"net/http"
)

// DoWithContext 在执行 HTTP 请求前，将 context 中的 request_id 注入 X-Request-ID 请求头，
// 实现出站 HTTP 调用的 request_id 传播，保证跨服务链路可追踪。
//
// 若 context 中不存在 request_id，则不写入头部（不注入空字符串），也不 panic。
// req.Header 可为 nil（函数会按需初始化）。
//
// 典型用法（服务层调用外部 API 时）：
//
//	resp, err := obs.DoWithContext(ctx, httpClient, req)
//
// 注意：DoWithContext 不修改原始 *http.Request（根据 Go 惯例，调用方持有 req 所有权）。
// 若 X-Request-ID 已存在，会被覆盖（以 ctx 中的值为准）。
func DoWithContext(ctx context.Context, c *http.Client, req *http.Request) (*http.Response, error) {
	if id, ok := RequestIDFromContext(ctx); ok {
		if req.Header == nil {
			req.Header = make(http.Header)
		}
		req.Header.Set("X-Request-ID", id)
	}
	return c.Do(req)
}
