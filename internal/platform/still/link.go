package still

import (
	"fmt"
	"hash/fnv"
)

// PersonLink is where a person's uploaded picture is served, versioned by its content so no cache serves an old one.
func PersonLink(userID, picture string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(picture)) // a hash.Hash never returns a write error
	return fmt.Sprintf("/api/people/%s/avatar?v=%x", userID, h.Sum64())
}
