// Package httputil provides HTTP utility functions including retry logic.
package httputil

import (
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	defaultRetryInitialInterval = 1 * time.Second
	defaultRetryMultiplier      = 2.0
	defaultRetryMaxInterval     = 4 * time.Second
	defaultRetryMaxTries        = 4
)

// RetryConfig defines retry/backoff tuning parameters.
type RetryConfig struct {
	InitialInterval time.Duration
	Multiplier      float64
	MaxInterval     time.Duration
	MaxTries        uint
}

// RetryOpts builds retry options from the provided config.
func RetryOpts(cfg RetryConfig) []backoff.RetryOption {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = cfg.InitialInterval
	expBackoff.Multiplier = cfg.Multiplier
	expBackoff.MaxInterval = cfg.MaxInterval

	return []backoff.RetryOption{
		backoff.WithBackOff(expBackoff),
		backoff.WithMaxTries(cfg.MaxTries),
	}
}

// DefaultRetryOpts returns the standard retry options used across HTTP clients.
func DefaultRetryOpts() []backoff.RetryOption {
	return RetryOpts(RetryConfig{
		InitialInterval: defaultRetryInitialInterval,
		Multiplier:      defaultRetryMultiplier,
		MaxInterval:     defaultRetryMaxInterval,
		MaxTries:        defaultRetryMaxTries,
	})
}

// IsPermanentHTTPError returns true if the HTTP status code represents a
// permanent (non-retriable) error. Transient errors like 408 (Request Timeout)
// and 429 (Too Many Requests) are retriable; all other 4xx are permanent.
func IsPermanentHTTPError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500 && statusCode != 408 && statusCode != 429
}
