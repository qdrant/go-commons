// Package errors introduces utilities to wrap error with additional metadata
// so that the error has proper context when it is logged.
// The idea was borrowed from: https://medium.com/@oberonus/context-matters-advanced-error-handling-techniques-in-go-b470f763c7ec
package errors

import (
	"errors"
)

// errWithMetadata represents an error with attached metadata
type errWithMetadata struct {
	// err is the original error
	err error
	// metadata is the container for error context
	metadata []string
}

// Error returns the original error message,
// ensuring compatibility with the standard error interface.
func (w *errWithMetadata) Error() string {
	return w.err.Error()
}

// Unwrap returns the original error that was wrapped with errWithMetadata instance
// It makes the errWithMetadata compatible with the standard error unwrapping mechanism
func (w *errWithMetadata) Unwrap() error {
	return w.err
}

type errorMetadata []string

// Metadata returns a new metadata container with the provided key value pairs
func Metadata(keyValues ...string) errorMetadata {
	return keyValues
}

// Extend returns a new metadata container with combined key value pairs from current metadata and provided key value pairs
func (m *errorMetadata) Extend(keyValues ...string) errorMetadata {
	if m == nil {
		return keyValues
	}
	return mergeKeyValuePair(*m, keyValues)
}

// WithMetadata returns the provided error wrapped with the provided metadata
func WithMetadata(err error, metadata ...string) error {
	if err == nil {
		return nil
	}
	return &errWithMetadata{
		err:      err,
		metadata: metadata,
	}
}

// GetMetadata returns metadata from the error chain
// If there is no metadata in the chain, it will return an empty slice
// It returns []any to make it compatible with structured logging libraries
func GetMetadata(err error) []any {
	metadata := make([]string, 0)

	// We will iterate over all errors in the chain
	// and merge metadata from all of them
	for err != nil {
		// If current error is wrapped with our errWithMetadata,
		// we will add its metadata to our metadata store
		if e, ok := err.(*errWithMetadata); ok { // nolint: errorlint
			metadata = append(metadata, e.metadata...)
		}
		// move to the next error in the chain
		err = errors.Unwrap(err)
	}

	// convert []string to []any
	result := make([]any, 0)
	for i := range metadata {
		result = append(result, metadata[i])
	}
	return result
}

// mergeKeyValuePair merges two slices into a new slice.
// It assumes that both slices are valid key value pairs.
// If a key is missing a value, it will add a padding "<missing>" to the slice.
func mergeKeyValuePair(cur, new []string) []string {
	// Both "cur" and "new" should be valid key value pair.
	// We will be adding a padding in case some key misses value.
	paddedCur := addPaddingForMissingValue(cur)
	paddedNew := addPaddingForMissingValue(new)
	// just to avoid reallocation, we will create a new slice with the combined length
	newKV := make([]string, 0, len(paddedCur)+len(paddedNew))
	newKV = append(newKV, paddedCur...)
	newKV = append(newKV, paddedNew...)
	return newKV
}

// addPaddingForMissingValue adds a padding "<missing>" to the slice if the last key is missing a value
func addPaddingForMissingValue(keyValues []string) []string {
	newLen := len(keyValues)
	// check if the last key has a value
	missingValue := len(keyValues)%2 != 0
	if missingValue {
		newLen++
	}

	// create a new slice with the new length
	newKV := make([]string, 0, newLen)
	// copy the key values to the new slice
	newKV = append(newKV, keyValues...)
	// add padding if the last key is missing a value
	if missingValue {
		newKV = append(newKV, "<missing>")
	}
	return newKV
}

// nonRetryableError is a wrapper for errors that resulted from an operation that should not be retried
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

// AsNonRetryableError wraps an error as non-retryable error
func AsNonRetryableError(err error, metadata ...string) error {
	return &nonRetryableError{err: WithMetadata(err, metadata...)}
}

// IsNonRetryableError checks if the error is a non-retryable or not
func IsNonRetryableError(err error) bool {
	var e *nonRetryableError
	return errors.As(err, &e)
}

// retryableError is a wrapper for errors that resulted from an operation that can be retried
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

// AsRetryableError wraps an error as retryable error
func AsRetryableError(err error, metadata ...string) error {
	return &retryableError{err: WithMetadata(err, metadata...)}
}

// IsRetryableError checks if the error is retryable or not
func IsRetryableError(err error) bool {
	var e *retryableError
	return errors.As(err, &e)
}
