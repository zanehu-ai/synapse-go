# synapse-go

Go 共享基础设施库。为所有 Go 产品提供统一的基础设施层（L1）和平台能力层（L2）。

## 安装

```bash
# 配置私有模块访问（如仓库为 private）
export GOPRIVATE=github.com/techfitmaster/*

go get github.com/techfitmaster/synapse-go@latest
```

## 包列表

### L1 — 基础设施

| 包 | 说明 |
|---|------|
| `config` | 配置结构体（MySQL/Redis/Auth/SMTP）+ `GetEnv()` 环境变量加载 |
| `db` | MySQL 连接池初始化（GORM） |
| `redis` | Redis 连接初始化 |
| `logger` | 结构化日志（zap） |
| `migrate` | 数据库迁移（golang-migrate） |
| `timeutil` | 时间工具函数 |
| `mailer` | 邮件发送（SMTP + NoopMailer） |

### L2 — 平台能力

| 包 | 说明 |
|---|------|
| `resp` | 统一 API 响应格式（Success/Error/SuccessPage + 错误码） |
| `bizerr` | 业务错误类型（BizError + HandleError + 便捷构造函数） |
| `ginutil` | Gin 工具（路由参数解析 + Context Helpers） |
| `middleware` | HTTP 中间件（CORS、RequestID、JWT、角色鉴权、Header Secret） |
| `ratelimit` | 多维限流（RPM/TPM/并发，Redis Lua 脚本） |
| `crypto` | 密码哈希（bcrypt）+ 验证码生成 + 密码强度校验 |
| `lock` | Redis 分布式锁（SetNX + token 所有权 + Lua 原子释放） |
| `audit` | 审计日志（Gin 中间件自动记录 + 事务内手动记录） |
| `idempotent` | 幂等控制（Idempotency-Key 中间件 + Service 层 Check） |
| `sequence` | Redis 序列号生成（日期分区 + 自动递增） |
| `notify` | 通知抽象层（邮件/Webhook + 重试包装器） |
| `validate` | 通用校验（邮箱/手机号格式） |
| `healthcheck` | 健康检查（多检查项注册 + Gin Handler） |
| `scheduler` | 定时任务调度（后台 goroutine + 分布式锁） |
| `storage` | 对象存储抽象（本地文件系统 / 可扩展 OSS） |

### L3 — 架构能力

| 包 | 说明 |
|---|------|
| `circuitbreaker` | 熔断器（Closed/Open/HalfOpen 状态机） |
| `tenant` | 多租户上下文（Gin 中间件 + context 传递） |
| `event` | 轻量事件总线（同步/异步发布 + panic 恢复） |
| `cache` | 缓存抽象（Redis 实现 + GetOrLoad 模式） |
| `graceful` | HTTP Server 优雅关停（信号监听 + 超时控制） |

## 使用示例

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/techfitmaster/synapse-go/config"
    "github.com/techfitmaster/synapse-go/db"
    "github.com/techfitmaster/synapse-go/logger"
    "github.com/techfitmaster/synapse-go/resp"
    "github.com/techfitmaster/synapse-go/middleware"
)

func main() {
    cfg := config.Load()
    log := logger.New(cfg.Env)
    gormDB, _ := db.New(cfg.MySQL)

    r := gin.New()
    r.Use(middleware.RequestIDMiddleware())
    r.Use(middleware.CORSMiddleware("http://localhost:3000"))

    r.GET("/health", func(c *gin.Context) {
        resp.Success(c, gin.H{"status": "ok"})
    })
}
```

## API 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... },
  "trace_id": "abc-123"
}
```

错误码规范：`0` 成功 / `1xxx` 客户端错误 / `2xxx` 鉴权错误 / `5xxx` 服务端错误

## 开发

```bash
make test          # 运行单元测试
make test-verbose  # 详细输出
make lint          # golangci-lint
make coverage      # 生成覆盖率报告
```

## 版本管理

推送 tag 发布新版本：
```bash
git tag v0.3.0
git push origin v0.3.0
```

消费方更新：
```bash
go get github.com/techfitmaster/synapse-go@v0.3.0
```
