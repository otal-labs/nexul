package integrations

import "github.com/otal-labs/nexul/internal/platform/webhooksig"

// VerifyWebhookSignature is how a receiver checks the header a delivery carries.
func VerifyWebhookSignature(secret string, payload []byte, header string) bool {
	return webhooksig.Verify(secret, payload, header)
}
