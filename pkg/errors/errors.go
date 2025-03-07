// Package errors introduces utilities to wrap error with additional metadata
// so that the error has proper context when it is logged.
// The idea was borrowed from: https://medium.com/@oberonus/context-matters-advanced-error-handling-techniques-in-go-b470f763c7ec
package errors

import (
	"errors"
)

// errWrapper represents an error with attached metadata
type errWrapper struct {
	// err is the original error
	err error
	// metadata is the container for error context
	metadata []any
}

// Error returns the original error message,
// ensuring compatibility with the standard error interface.
func (w *errWrapper) Error() string {
	return w.err.Error()
}

// Unwrap returns the original error that was wrapped with errWrapper instance
// It makes the errWrapper compatible with the standard error unwrapping mechanism
func (w *errWrapper) Unwrap() error {
	return w.err
}

// GetMetadata returns metadata from the error chain
// If there is no metadata in the chain, it will return an empty slice
func GetMetadata(err error) []any {
	metadata := make([]any, 0)

	// We will iterate over all errors in the chain
	// and merge metadata from all of them
	for err != nil {
		// If current error is wrapped with our errWrapper,
		// we will add its metadata to our metadata store
		if e, ok := err.(*errWrapper); ok {
			// merge e.metadata into metadata
			metadata = append(metadata, e.metadata...)
		}
		// move to the next error in the chain
		err = errors.Unwrap(err)
	}
	return metadata
}

// Context creates a new errWrapper with the provided metadata
func Context(keyValues ...any) *errWrapper {
	return &errWrapper{metadata: keyValues}
}

// Extend creates a new errWrapper with combined metadata from the current errWrapper and the provided metadata
func (w *errWrapper) Extend(keyValues ...any) *errWrapper {
	return &errWrapper{metadata: mergeKeyValuePair(w.metadata, keyValues)}
}

// With returns the provided error wrapped with metadata from current wrapper
func (w *errWrapper) With(err error) error {
	if err == nil || w == nil {
		return nil
	}
	return &errWrapper{err: err, metadata: w.metadata}
}

// WithMetadata returns the provided error wrapped with the provided metadata combined with metadata from current wrapper
func (w *errWrapper) WithMetadata(err error, metadata ...any) error {
	if err == nil || w == nil {
		return nil
	}
	return &errWrapper{
		err:      err,
		metadata: mergeKeyValuePair(w.metadata, metadata),
	}
}

// WithMetadata returns the provided error wrapped with the provided metadata
func WithMetadata(err error, metadata ...any) error {
	w := errWrapper{}
	return w.WithMetadata(err, metadata...)
}

// mergeKeyValuePair merges two slices into a new slice. It assumes that both slices are valid key value pairs.
// If a key is missing a value, it will add a padding "<missing>" to the slice.
func mergeKeyValuePair(cur, new []any) []any {
	// Both "cur" and "new" should be valid key value pair.
	// We will be adding a padding in case some key misses value.
	newLen := len(cur) + len(new)

	// Check if the "cur" and "new" have missing value
	curMissingValue := len(cur)%2 != 0
	if curMissingValue {
		newLen++
	}
	newMissingValue := len(new)%2 != 0
	if newMissingValue {
		newLen++
	}

	// Create a new slice so that we don't modify the original one
	newKV := make([]any, 0, newLen)
	// Add padding if "cur" has missing value
	newKV = append(newKV, cur...)
	if curMissingValue {
		newKV = append(newKV, "<missing>")
	}
	// Add padding if "new" has missing value
	newKV = append(newKV, new...)
	if newMissingValue {
		newKV = append(newKV, "<missing>")
	}
	return newKV
}

// nonRetryableError is a errWrapper for errors that resulted from an operation that should not be retried
type nonRetryableError struct {
	err error
}

// Error returns the original error message,
func (e *nonRetryableError) Error() string {
	return e.err.Error()
}

// Unwrap returns the original error.
// It allows the error to be compatible with standard error unwrapping mechanism
func (e *nonRetryableError) Unwrap() error {
	return e.err
}

// AsNonRetryableError wraps an error with nonRetryableError
func AsNonRetryableError(err error, metadata ...any) error {
	return &nonRetryableError{err: WithMetadata(err, metadata...)}
}

// IsNonRetryableError checks if the error is a nonRetryableError
func IsNonRetryableError(err error) bool {
	var e *nonRetryableError
	return errors.As(err, &e)
}

// retryableError is a errWrapper for errors that resulted from an operation that can be retried
type retryableError struct {
	err error
}

// Error returns the original error message,
func (e *retryableError) Error() string {
	return e.err.Error()
}

// Unwrap returns the original error.
// It allows the error to be compatible with standard error unwrapping mechanism
func (e *retryableError) Unwrap() error {
	return e.err
}

// AsRetryableError wraps an error with retryableError
func AsRetryableError(err error, metadata ...any) error {
	return &retryableError{err: WithMetadata(err, metadata...)}
}

// IsRetryableError checks if the error is a retryableError
func IsRetryableError(err error) bool {
	var e *retryableError
	return errors.As(err, &e)
}
