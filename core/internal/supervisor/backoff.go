package supervisor

import "time"

// backoffDuration returns the delay before the next restart attempt.
// Schedule: 1s, 2s, 4s, 8s, 16s (exponential with 2x base).
func backoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	d := time.Duration(1<<attempt) * time.Second // 2^attempt seconds
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}