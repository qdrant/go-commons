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

type Metadata []string

// Extend returns a new metadata container with combined key value pairs from current metadata and provided key value pairs
func (m *Metadata) Extend(keyValues ...string) Metadata {
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
