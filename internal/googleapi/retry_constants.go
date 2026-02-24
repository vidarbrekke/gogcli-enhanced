package googleapi

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// MaxRateLimitRetries is the maximum number of retries on 429 responses.
	MaxRateLimitRetries = 3
	// RateLimitBaseDelay is the initial delay for rate limit exponential backoff.
	RateLimitBaseDelay = 1 * time.Second
	// Max5xxRetries is the maximum retries for server errors.
	Max5xxRetries = 1
	// ServerErrorRetryDelay is the delay before retrying on 5xx errors.
	ServerErrorRetryDelay = 1 * time.Second
)

var runtimeRetryConfig = struct {
	MaxRetries429 int
	MaxRetries5xx int
	BaseDelay     time.Duration
}{
	MaxRetries429: MaxRateLimitRetries,
	MaxRetries5xx: Max5xxRetries,
	BaseDelay:     RateLimitBaseDelay,
}

var errRetryBackoffNonPositive = errors.New("retry-backoff must be > 0")

// ConfigureRetryPolicy overrides runtime retry policy for API transports.
// Pass retries < 0 to keep defaults. Empty backoff keeps default backoff.
func ConfigureRetryPolicy(retries int, backoff string) error {
	if retries >= 0 {
		runtimeRetryConfig.MaxRetries429 = retries
		runtimeRetryConfig.MaxRetries5xx = retries
	}

	if strings.TrimSpace(backoff) != "" {
		d, err := time.ParseDuration(backoff)
		if err != nil {
			return fmt.Errorf("parse retry-backoff: %w", err)
		}

		if d <= 0 {
			return errRetryBackoffNonPositive
		}

		runtimeRetryConfig.BaseDelay = d
	}

	return nil
}
