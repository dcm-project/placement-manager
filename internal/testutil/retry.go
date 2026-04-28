// Package testutil provides shared helpers used in tests.
package testutil

import (
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/dcm-project/placement-manager/internal/httputil"
)

// FastRetryOpts returns a short retry profile intended for unit tests.
func FastRetryOpts() []backoff.RetryOption {
	return httputil.RetryOpts(httputil.RetryConfig{
		InitialInterval: 5 * time.Millisecond,
		Multiplier:      1.5,
		MaxInterval:     20 * time.Millisecond,
		MaxTries:        2,
	})
}
