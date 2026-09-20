// Package ids generates the time-sortable UUIDv7 identifiers domain
// use-cases mint for new rows.
package ids

import "github.com/google/uuid"

// New returns a fresh UUIDv7 (time-sortable). Falls back to a random UUIDv4
// on the vanishingly rare NewV7 error (clock read failure).
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
