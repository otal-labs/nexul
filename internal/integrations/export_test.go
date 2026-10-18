package integrations

import "crypto/hmac"

// VerifyWebhookSignature is how a receiver checks the header signWebhook produced.
func VerifyWebhookSignature(secret string, payload []byte, header string) bool {
	return hmac.Equal([]byte(header), []byte(signWebhook(secret, payload)))
}
