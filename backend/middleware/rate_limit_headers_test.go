package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		name      string
		remaining time.Duration
		want      int
	}{
		{"sub-second wait is never advertised as zero", 400 * time.Millisecond, 1},
		{"already-reset window still advertises at least 1", 0, 1},
		{"negative remaining is clamped to 1", -2 * time.Second, 1},
		{"whole seconds are returned unchanged", 30 * time.Second, 30},
		{"fractional seconds round up so clients never retry early", 1500 * time.Millisecond, 2},
		{"full window", time.Minute, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, retryAfterSeconds(tt.remaining))
		})
	}
}
