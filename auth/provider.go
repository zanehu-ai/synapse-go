// Package auth — ProviderAdapter defines the interface every external OAuth /
// SSO provider must implement to participate in Synapse's federated login
// flows.
//
// # Credential namespace convention
//
// Every provider integration stores exactly one credential row in the
// `credential` table per (provider, tenant, external_id) triple. The
// Identifier column follows a three-segment namespace:
//
//	<provider_name>:<tenant_code>:<external_id>
//
// Examples:
//   - wechat:dnf-cn:oXXXXXXXXXXXXXXXXXXXXXXX
//   - wechat:dnf-kr:oYYYYYYYYYYYYYYYYYYYYYYY   ← same openid, different tenant
//   - dingtalk:buycn:user_ZZZZZ
//
// Per-tenant namespacing means the same external_id (e.g. the same WeChat
// openid) maps to distinct credential rows under different tenants. This
// is intentional: the same physical person may be a player in dnf-cn *and*
// dnf-kr with independent game accounts; collapsing them would break
// tenant isolation (ADR-001 §5) and expose cross-tenant session derivation.
package auth

import "context"

// ProviderAdapter is the abstraction layer that every external OAuth /
// identity provider (WeChat, DingTalk, GitHub, VK, …) implements to
// plug into Synapse's federated login flow.
//
// Implementations live in apps/control/internal/identity/ (platform-level)
// or in a template's own package (template-specific providers). The
// interface intentionally carries no HTTP dependencies — callers inject
// their own http.Client so tests can swap in httptest.Server mocks.
type ProviderAdapter interface {
	// Name returns the canonical lowercase provider name used as the first
	// segment of the credential identifier namespace, e.g. "wechat",
	// "dingtalk", "github". Must be stable across deployments — changing it
	// orphans existing credential rows.
	Name() string

	// ExchangeCode trades the short-lived login code (e.g. WeChat jscode,
	// OAuth 2 authorization code) for an openid and an access token.
	//
	// tenantCode identifies which tenant's App config (appID / appSecret)
	// to use for the exchange. Implementations MUST NOT fall back to a
	// different tenant's credentials when the requested tenant config is
	// absent — they must return ErrProviderTenantNotConfigured instead.
	//
	// The returned openid is the stable, provider-issued external user
	// identifier that becomes the third segment of the credential namespace.
	// The returned accessToken is provider-specific: for WeChat mini-programs
	// it is the session_key (which is NOT an OAuth access_token), for OAuth 2
	// flows it is the bearer token to pass to FetchUserInfo.
	ExchangeCode(ctx context.Context, tenantCode, code, state string) (openid, accessToken string, err error)

	// FetchUserInfo retrieves display-name and avatar URL from the provider
	// using the access token returned by ExchangeCode.
	//
	// For providers that do not support server-side profile fetching (e.g.
	// WeChat mini-program jscode2session — user profile is collected client-
	// side via wx.getUserProfile and then uploaded by the app), FetchUserInfo
	// SHOULD return ("", "", nil) as a no-op placeholder. Callers must treat
	// empty nickname / avatarURL as "profile not available server-side" and
	// accept client-uploaded profile separately.
	FetchUserInfo(ctx context.Context, accessToken string) (nickname, avatarURL string, err error)
}
