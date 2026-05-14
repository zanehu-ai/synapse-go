// Package pii implements the multi-jurisdiction PII redaction layer per
// ADR-010 §6.2 / §6.3. V1 scope: RU + CN + EN universal rules. Both
// inbound (user message → LLM) and outbound (LLM → user) directions
// must be scrubbed; outbound scrubbing prevents the model from
// hallucinating real PII into responses.
//
// The redactor uses precompiled regex tables — never fuzzy / heuristic
// matching (§6.3 explicit rule). Adding a new PII type means adding a
// new entry to one of the rule tables and a corresponding test.
//
// CI must enforce >= 95% UT coverage on this package (§6.4).
package pii

import (
	"regexp"
	"strings"
)

// Jurisdiction codes used to select which regex set runs first. Universal
// rules always run regardless. The redactor accepts a hint and falls back
// to "all jurisdictions" when the hint is empty / unknown.
const (
	JurisdictionRU = "ru"
	JurisdictionCN = "cn"
	JurisdictionEN = "en"
)

// rule pairs a regex with a redaction function. The function receives
// the raw match and returns its replacement.
type rule struct {
	pattern *regexp.Regexp
	apply   func(match string) string
	label   string // for tests / debug
}

// universalRules apply regardless of jurisdiction (tokens / keys / etc).
var universalRules = []rule{
	// JWT-ish tokens (3 base64url segments separated by dots, each at
	// least 16 chars to skip non-token dot patterns).
	{
		pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{16,}\.[A-Za-z0-9_\-]{16,}\.[A-Za-z0-9_\-]{16,}\b`),
		apply:   func(_ string) string { return "[REDACTED:JWT]" },
		label:   "jwt",
	},
	// Bearer / API key prefixes commonly seen in support pastes.
	// Body allows alnum + underscore so "sk_live_AbCdEfGh..." matches.
	{
		pattern: regexp.MustCompile(`\b(?i:sk|pk|api[_-]?key)[_-][A-Za-z0-9_]{16,}\b`),
		apply:   func(_ string) string { return "[REDACTED:APIKEY]" },
		label:   "api_key",
	},
	// Bank card numbers (13-19 digits with optional spaces / dashes).
	// Common Visa/MC/UnionPay/Mir lengths covered by 13-19. Keep first 4
	// + last 4 visible per §6.2.
	{
		pattern: regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{1,7}\b`),
		apply:   redactBankCard,
		label:   "bank_card",
	},
	// Email addresses. a***@host.tld per §6.2.
	{
		pattern: regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
		apply:   redactEmail,
		label:   "email",
	},
	// IPv4 → /16 (keep first two octets, mask last two).
	{
		pattern: regexp.MustCompile(`\b(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\b`),
		apply:   redactIPv4,
		label:   "ipv4",
	},
}

// ruRules cover Russian PII formats.
var ruRules = []rule{
	// Russian mobile: +7-9XX-XXX-XX-XX or 89XXXXXXXXX (and variants
	// with parens / spaces / dashes).
	{
		pattern: regexp.MustCompile(`(?:\+?7|8)[\s\-\(]?9\d{2}[\s\-\)]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}`),
		apply:   redactRUPhone,
		label:   "ru_phone",
	},
	// SNILS: XXX-XXX-XXX YY (3+3+3+2).
	{
		pattern: regexp.MustCompile(`\b\d{3}[\-\s]\d{3}[\-\s]\d{3}[\-\s]\d{2}\b`),
		apply:   func(_ string) string { return "[REDACTED:SNILS]" },
		label:   "snils",
	},
	// INN (russian taxpayer): 10 digits (entity) or 12 digits (individual).
	{
		pattern: regexp.MustCompile(`\b\d{10,12}\b`),
		apply:   redactINNCandidate,
		label:   "inn",
	},
}

// cnRules cover Chinese mainland PII formats.
var cnRules = []rule{
	// CN mobile: 1[3-9]XXXXXXXXX (11 digits starting with 1, 2nd digit 3-9).
	{
		pattern: regexp.MustCompile(`\b1[3-9]\d{9}\b`),
		apply:   redactCNPhone,
		label:   "cn_phone",
	},
	// CN national ID: 18 digits with optional X at end.
	{
		pattern: regexp.MustCompile(`\b\d{17}[\dXx]\b`),
		apply:   func(_ string) string { return "[REDACTED:CN_ID]" },
		label:   "cn_id",
	},
}

// Redactor applies the rule tables. Construct via NewRedactor.
type Redactor struct {
	universal []rule
	ru        []rule
	cn        []rule
}

// NewRedactor returns a default-configured Redactor. Future opts can
// disable specific rule sets per tenant configuration.
func NewRedactor() *Redactor {
	return &Redactor{
		universal: universalRules,
		ru:        ruRules,
		cn:        cnRules,
	}
}

// RedactString redacts PII in s. languageOrJurisdiction can be a
// jurisdiction code ("ru" / "cn" / "en") or a language hint that maps
// to one ("en-US" → en, etc). Empty / unknown values run all rule sets
// — the conservative choice.
//
// Rule ordering: jurisdiction-specific rules run FIRST (so CN IDs get a
// chance to match before the universal bank-card rule swallows their
// 18-digit form). Universal rules run AFTER on whatever's left.
func (r *Redactor) RedactString(languageOrJurisdiction, s string) string {
	if s == "" {
		return s
	}
	hint := strings.ToLower(strings.TrimSpace(languageOrJurisdiction))
	if idx := strings.IndexAny(hint, "_-"); idx > 0 {
		hint = hint[:idx]
	}
	out := s
	switch hint {
	case JurisdictionRU:
		for _, rl := range r.ru {
			out = applyRule(out, rl)
		}
	case JurisdictionCN:
		for _, rl := range r.cn {
			out = applyRule(out, rl)
		}
	case JurisdictionEN:
		// No jurisdiction-specific rules — universal-only.
	case "":
		// Empty jurisdiction → run BOTH RU and CN rule sets.
		for _, rl := range r.ru {
			out = applyRule(out, rl)
		}
		for _, rl := range r.cn {
			out = applyRule(out, rl)
		}
	default:
		// Unknown jurisdiction → run BOTH RU and CN rule sets.
		for _, rl := range r.ru {
			out = applyRule(out, rl)
		}
		for _, rl := range r.cn {
			out = applyRule(out, rl)
		}
	}
	for _, rl := range r.universal {
		out = applyRule(out, rl)
	}
	return out
}

func applyRule(s string, r rule) string {
	return r.pattern.ReplaceAllStringFunc(s, r.apply)
}

func redactEmail(match string) string {
	at := strings.IndexByte(match, '@')
	if at < 1 {
		return "[REDACTED:EMAIL]"
	}
	local := match[:at]
	domain := match[at+1:]
	keep := local[:1]
	return keep + "***@" + domain
}

func redactBankCard(match string) string {
	digits := keepDigits(match)
	if len(digits) < 13 || len(digits) > 19 {
		return match // not actually a bank card; leave alone
	}
	return digits[:4] + " **** **** " + digits[len(digits)-4:]
}

func redactIPv4(match string) string {
	parts := strings.Split(match, ".")
	if len(parts) != 4 {
		return match
	}
	for _, p := range parts {
		if len(p) < 1 || len(p) > 3 {
			return match
		}
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return match
			}
		}
	}
	return parts[0] + "." + parts[1] + ".*.*"
}

func redactRUPhone(match string) string {
	digits := keepDigits(match)
	if len(digits) < 11 {
		return match
	}
	// Keep country code + first three of the operator code, mask the rest
	// except last two digits per §6.2.
	if digits[0] == '8' && len(digits) == 11 {
		// Convert 89XXXXXXXXX → +7-9XX-***-**-XX form
		return "+7-" + digits[1:4] + "-***-**-" + digits[len(digits)-2:]
	}
	if digits[0] == '7' && len(digits) >= 11 {
		return "+7-" + digits[1:4] + "-***-**-" + digits[len(digits)-2:]
	}
	return "+7-***-***-**-" + digits[len(digits)-2:]
}

func redactCNPhone(match string) string {
	digits := keepDigits(match)
	if len(digits) != 11 {
		return match
	}
	return digits[:3] + "****" + digits[len(digits)-4:]
}

// redactINNCandidate fires only when the digit string length is exactly
// 10 or 12 — INNs have those exact lengths. Length-13+ matches are bank
// cards (handled by the universal rule earlier in the chain) and length
// <10 are treated as ordinary numbers.
func redactINNCandidate(match string) string {
	if len(match) == 10 || len(match) == 12 {
		return "[REDACTED:INN]"
	}
	return match
}

func keepDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
