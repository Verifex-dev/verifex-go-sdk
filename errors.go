package verifex

import "fmt"

// APIError is the base error returned by the Verifex API.
type APIError struct {
	Message    string `json:"error"`
	Code       string `json:"code"`
	StatusCode int    `json:"-"`
	RequestID  string `json:"request_id"`
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("verifex [%s]: %s (request_id=%s)", e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("verifex [%s]: %s", e.Code, e.Message)
}

// AuthenticationError is returned for 401 responses.
type AuthenticationError struct {
	APIError
}

// RateLimitError is returned for 429 responses.
type RateLimitError struct {
	APIError
	RetryAfter int `json:"-"`
}

// QuotaExceededError is returned for 402 responses.
type QuotaExceededError struct {
	APIError
}

// IsAuthError reports whether the error is an authentication failure.
func IsAuthError(err error) bool {
	_, ok := err.(*AuthenticationError)
	return ok
}

// IsRateLimitError reports whether the error is a rate limit failure.
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// IsQuotaExceededError reports whether the error is a quota exceeded failure.
func IsQuotaExceededError(err error) bool {
	_, ok := err.(*QuotaExceededError)
	return ok
}
