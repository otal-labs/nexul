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

// derived namespaces From's ids, so they never collide with a UUID minted any other way.
var derived = uuid.MustParse("6f0b8a52-0c4e-4f43-9d0f-2b7c1f6e9a31")

// From returns the same UUIDv5 for the same key every time, for a row that must be written at most once per key.
func From(key string) string {
	return uuid.NewSHA1(derived, []byte(key)).String()
}
