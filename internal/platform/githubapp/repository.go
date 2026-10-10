package githubapp

import (
	"fmt"
	"regexp"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var repositorySegment = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ValidateRepository prevents URL normalization from changing the account whose installation is authorized.
func ValidateRepository(owner, name string) error {
	for _, segment := range []string{owner, name} {
		if segment == "." || segment == ".." || !repositorySegment.MatchString(segment) {
			return fmt.Errorf("%w: owner and name must be single repository path segments using letters, numbers, dots, underscores or hyphens", apperrs.ErrInvalid)
		}
	}
	return nil
}
