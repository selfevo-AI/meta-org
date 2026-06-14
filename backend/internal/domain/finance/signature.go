package finance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func SignPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func VerifyPayload(body []byte, signature string, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}
	expected := SignPayload(body, secret)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}

func BatchIdempotencyKey(adapterID, periodStart, periodEnd, currency string) string {
	payload := strings.Join([]string{
		strings.TrimSpace(adapterID),
		strings.TrimSpace(periodStart),
		strings.TrimSpace(periodEnd),
		strings.ToUpper(strings.TrimSpace(currency)),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
