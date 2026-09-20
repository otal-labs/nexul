package tenancy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func HashInvitationToken(raw string) (string, error) {
	token, err := uuid.Parse(raw)
	if err != nil || token.Version() != 4 {
		return "", fmt.Errorf("%w: invalid invitation token", apperrs.ErrInvalid)
	}
	sum := sha256.Sum256([]byte(token.String()))
	return hex.EncodeToString(sum[:]), nil
}
