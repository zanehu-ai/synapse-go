package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	defaultTOTPSecretBytes = 20
	defaultTOTPPeriod      = 30 * time.Second
	defaultTOTPDigits      = 6
)

// GenerateTOTPSecret returns a random base32 secret suitable for TOTP.
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, defaultTOTPSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), "="), nil
}

// TOTPCode returns the TOTP code for secret at now using SHA-1, 30s period,
// and 6 digits.
func TOTPCode(secret string, now time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return ""
	}
	counter := uint64(now.Unix() / int64(defaultTOTPPeriod.Seconds()))
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	mod := uint32(math.Pow10(defaultTOTPDigits))
	return fmt.Sprintf("%0*d", defaultTOTPDigits, bin%mod)
}

// ValidateTOTP verifies code against secret at now, allowing windowSteps
// periods of clock skew on either side.
func ValidateTOTP(secret, code string, now time.Time, windowSteps int) bool {
	if windowSteps < 0 {
		windowSteps = 0
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for offset := -windowSteps; offset <= windowSteps; offset++ {
		at := now.Add(time.Duration(offset) * defaultTOTPPeriod)
		if subtle.ConstantTimeCompare([]byte(TOTPCode(secret, at)), []byte(code)) == 1 {
			return true
		}
	}
	return false
}
