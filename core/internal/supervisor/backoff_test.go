package supervisor

import (
	"testing"
	"time"
)

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // capped
		{10, 30 * time.Second}, // capped
	}

	for _, tt := range tests {
		got := backoffDuration(tt.attempt)
		if got != tt.expected {
			t.Errorf("backoffDuration(%d) = %v, want %v", tt.attempt, got, tt.expected)
		}
	}
}