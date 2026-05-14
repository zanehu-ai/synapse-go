package pii

import (
	"strings"
	"testing"
)

// Each test asserts the regex matches and the redaction transforms the
// input correctly. Group tests by jurisdiction to keep diff small.

func TestRedactor_Email(t *testing.T) {
	r := NewRedactor()
	cases := []struct {
		in, want string
	}{
		{"contact me at john.doe@example.com please", "contact me at j***@example.com please"},
		{"a@b.co edge", "a***@b.co edge"},
		{"no email here", "no email here"},
	}
	for _, c := range cases {
		got := r.RedactString("", c.in)
		if got != c.want {
			t.Errorf("RedactString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRedactor_BankCard(t *testing.T) {
	r := NewRedactor()
	cases := []struct {
		in, wantContains string
	}{
		{"card 4111 1111 1111 1111 thanks", "4111 **** **** 1111"},
		{"4111-1111-1111-1111", "4111 **** **** 1111"},
		{"unionpay 6212345678901234", "6212 **** **** 1234"},
	}
	for _, c := range cases {
		got := r.RedactString("", c.in)
		if !strings.Contains(got, c.wantContains) {
			t.Errorf("RedactString(%q) = %q, want contains %q", c.in, got, c.wantContains)
		}
	}
}

func TestRedactor_IPv4(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("", "client 192.168.1.42 connected")
	if !strings.Contains(got, "192.168.*.*") {
		t.Errorf("expected /16 redaction, got %q", got)
	}
}

func TestRedactor_JWT(t *testing.T) {
	r := NewRedactor()
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	got := r.RedactString("", "token: "+jwt)
	if !strings.Contains(got, "[REDACTED:JWT]") {
		t.Errorf("expected JWT redaction, got %q", got)
	}
}

func TestRedactor_APIKey(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("", "key sk_live_AbCdEfGh1234567890ZZ")
	if !strings.Contains(got, "[REDACTED:APIKEY]") {
		t.Errorf("expected api key redaction, got %q", got)
	}
}

func TestRedactor_RUPhone(t *testing.T) {
	r := NewRedactor()
	cases := []struct{ in, wantPrefix string }{
		{"+7 912 345 67 89", "+7-912-***-**-89"},
		{"contact 89123456789 now", "+7-912-***-**-89"},
		{"+7(912)3456789", "+7-912-***-**-89"},
	}
	for _, c := range cases {
		got := r.RedactString(JurisdictionRU, c.in)
		if !strings.Contains(got, c.wantPrefix) {
			t.Errorf("RedactString(ru, %q) = %q, want contains %q", c.in, got, c.wantPrefix)
		}
	}
}

func TestRedactor_SNILS(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString(JurisdictionRU, "SNILS 123-456-789 01")
	if !strings.Contains(got, "[REDACTED:SNILS]") {
		t.Errorf("expected SNILS redaction, got %q", got)
	}
}

func TestRedactor_INN(t *testing.T) {
	r := NewRedactor()
	// 12 digits → individual INN
	got := r.RedactString(JurisdictionRU, "INN 123456789012 here")
	if !strings.Contains(got, "[REDACTED:INN]") {
		t.Errorf("expected 12-digit INN redaction, got %q", got)
	}
	// 10 digits → entity INN
	got = r.RedactString(JurisdictionRU, "company 1234567890 ok")
	if !strings.Contains(got, "[REDACTED:INN]") {
		t.Errorf("expected 10-digit INN redaction, got %q", got)
	}
	// 9 digits → not INN
	got = r.RedactString(JurisdictionRU, "order 123456789 plain")
	if strings.Contains(got, "[REDACTED:INN]") {
		t.Errorf("9-digit number should not match INN, got %q", got)
	}
}

func TestRedactor_CNPhone(t *testing.T) {
	r := NewRedactor()
	cases := []struct{ in, want string }{
		{"please contact 13812345678 thanks", "please contact 138****5678 thanks"},
		{"19987654321", "199****4321"},
	}
	for _, c := range cases {
		got := r.RedactString(JurisdictionCN, c.in)
		if got != c.want {
			t.Errorf("RedactString(cn, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRedactor_CNNationalID(t *testing.T) {
	r := NewRedactor()
	cases := []string{
		"id 110105199003078813 here",
		"老身份证 11010519900307881X 末尾X",
	}
	for _, in := range cases {
		got := r.RedactString(JurisdictionCN, in)
		if !strings.Contains(got, "[REDACTED:CN_ID]") {
			t.Errorf("expected CN ID redaction in %q, got %q", in, got)
		}
	}
}

func TestRedactor_LanguageHintParsing(t *testing.T) {
	r := NewRedactor()
	// "ru-RU" should resolve to JurisdictionRU
	got := r.RedactString("ru-RU", "+7 912 345 67 89")
	if !strings.Contains(got, "+7-912-") {
		t.Errorf("expected ru rules to fire on ru-RU hint, got %q", got)
	}
	// "zh_CN" should resolve to JurisdictionCN
	got = r.RedactString("zh_CN", "13812345678")
	if !strings.Contains(got, "138****5678") {
		t.Errorf("expected cn rules to fire on zh_CN hint, got %q", got)
	}
}

func TestRedactor_UnknownJurisdictionAppliesAll(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("xx", "+7 912 345 67 89 and 13812345678")
	if !strings.Contains(got, "+7-912-***-**-89") {
		t.Errorf("expected RU rule on unknown jurisdiction, got %q", got)
	}
	if !strings.Contains(got, "138****5678") {
		t.Errorf("expected CN rule on unknown jurisdiction, got %q", got)
	}
}

func TestRedactor_EmptyInputUnchanged(t *testing.T) {
	r := NewRedactor()
	if r.RedactString("ru", "") != "" {
		t.Errorf("empty input should pass through unchanged")
	}
}

func TestRedactor_NoMatchUnchanged(t *testing.T) {
	r := NewRedactor()
	in := "Hello world this has no PII at all just plain words"
	if got := r.RedactString("en", in); got != in {
		t.Errorf("no-PII input should be unchanged, got %q", got)
	}
}

func TestRedactor_MultiplePIIInOneString(t *testing.T) {
	r := NewRedactor()
	in := "Email j@x.io, phone 13812345678, card 4111 1111 1111 1111"
	got := r.RedactString(JurisdictionCN, in)
	if !strings.Contains(got, "j***@x.io") {
		t.Errorf("expected email redaction, got %q", got)
	}
	if !strings.Contains(got, "138****5678") {
		t.Errorf("expected CN phone redaction, got %q", got)
	}
	if !strings.Contains(got, "4111 **** **** 1111") {
		t.Errorf("expected bank card redaction, got %q", got)
	}
}

func TestRedactor_BankCardOutsideRangeUnchanged(t *testing.T) {
	r := NewRedactor()
	// 12 digits → not a bank card (and would match INN if RU jurisdiction
	// active, which it isn't here — EN hint = universal only)
	in := "ref 123456789012 ok"
	got := r.RedactString(JurisdictionEN, in)
	if got != in {
		// Universal rules apply — but bank-card regex requires 13+, so this
		// shouldn't change. (If it does change, our regex is wrong.)
		// Allow change only if the regex pattern matched a 13-digit substring;
		// our input has 12 digits exactly, so we expect identity.
		t.Errorf("12-digit string should not be redacted by EN/universal-only, got %q", got)
	}
}

func TestKeepDigits(t *testing.T) {
	if got := keepDigits("+7 (912) 345-67-89"); got != "79123456789" {
		t.Errorf("keepDigits = %q", got)
	}
	if got := keepDigits("abc"); got != "" {
		t.Errorf("keepDigits non-numeric = %q", got)
	}
}

func TestRedactEmailMalformed(t *testing.T) {
	// Direct call with input regex would never produce, just exercises early return.
	if got := redactEmail("@example.com"); got != "[REDACTED:EMAIL]" {
		t.Errorf("expected fallback for missing local part, got %q", got)
	}
}

func TestRedactBankCardOutsideRange(t *testing.T) {
	// Direct call with 12 digits — function returns the input unchanged.
	in := "123456789012"
	if got := redactBankCard(in); got != in {
		t.Errorf("expected unchanged for 12-digit input, got %q", got)
	}
}

func TestRedactIPv4Malformed(t *testing.T) {
	// Direct invocation with shapes the regex would not produce, to
	// exercise the validation guards.
	cases := []string{"1.2.3", "1.2.3.4.5", "1.2.3.abc", "1.2.3.1234"}
	for _, in := range cases {
		if got := redactIPv4(in); got != in {
			t.Errorf("redactIPv4(%q) should be unchanged, got %q", in, got)
		}
	}
}

func TestRedactRUPhoneEdgeCases(t *testing.T) {
	// Short digits — must return unchanged.
	if got := redactRUPhone("12345"); got != "12345" {
		t.Errorf("expected unchanged for short input, got %q", got)
	}
	// Starts with neither 7 nor 8 — fall through to anonymous keep-last-2.
	got := redactRUPhone("99123456789")
	if !strings.HasSuffix(got, "89") {
		t.Errorf("expected last-2-digit keep, got %q", got)
	}
}

func TestRedactCNPhoneEdgeCases(t *testing.T) {
	// Wrong length → unchanged.
	if got := redactCNPhone("1381234567"); got != "1381234567" {
		t.Errorf("expected unchanged for 10-digit input, got %q", got)
	}
}

func TestRedactor_OutboundUseCase(t *testing.T) {
	// Simulates LLM hallucinating real-looking PII; outbound redaction
	// MUST scrub it before user-facing surface.
	r := NewRedactor()
	llmOut := "Sure, your contact is 13812345678 and your card 4111 1111 1111 1111."
	got := r.RedactString(JurisdictionCN, llmOut)
	if strings.Contains(got, "13812345678") {
		t.Errorf("CN phone leaked through outbound: %q", got)
	}
	if strings.Contains(got, "4111 1111 1111 1111") {
		t.Errorf("bank card leaked through outbound: %q", got)
	}
}

func TestRedactor_INNCandidateHelper(t *testing.T) {
	if got := redactINNCandidate("12345"); got != "12345" {
		t.Errorf("expected non-INN length unchanged, got %q", got)
	}
	if got := redactINNCandidate("1234567890"); got != "[REDACTED:INN]" {
		t.Errorf("expected 10-digit INN redacted, got %q", got)
	}
	if got := redactINNCandidate("123456789012"); got != "[REDACTED:INN]" {
		t.Errorf("expected 12-digit INN redacted, got %q", got)
	}
}

func TestRedactorEmptyHintRunsJurisdictionRules(t *testing.T) {
	r := NewRedactor()
	got := r.RedactString("", "SNILS 123-456-789 01 and phone 13812345678")
	if !strings.Contains(got, "[REDACTED:SNILS]") || !strings.Contains(got, "138****5678") {
		t.Fatalf("RedactString empty hint = %q, want RU and CN redaction", got)
	}
}
