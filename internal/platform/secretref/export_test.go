package secretref

import "time"

// SetReadTimeout shortens the bound on one read so a timeout can be tested without waiting for it.
func (c *Cache) SetReadTimeout(d time.Duration) { c.timeout = d }
