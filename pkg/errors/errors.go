// Package errors introduces utilities to wrap error with additional metadata
// so that the error has proper context when it is logged.
// The idea was borrowed from: https://medium.com/@oberonus/context-matters-advanced-error-handling-techniques-in-go-b470f763c7ec
package errors

import (
	"errors"
	"reflect"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// errWithMetadata represents an error with attached metadata
type errWithMetadata struct {
	// err is the original error
	err error
	// metadata is the container for error context
	metadata []any
}

// Error returns the original error message,
// ensuring compatibility with the standard error interface.
func (w *errWithMetadata) Error() string {
	return w.err.Error()
}

// GRPCStatus returns the gRPC status of the wrapped error, if it exists.
// This makes errWithMetadata compatible with gRPC's error handling,
// allowing it to preserve the original status code and message while
// carrying additional metadata.
// It achieves this by embedding the metadata into the status Details field
// as a protobuf Struct.
func (w *errWithMetadata) GRPCStatus() *status.Status {
	// Get the underlying status. If the wrapped error is not a gRPC status,
	// it will be converted to one with codes.Unknown.
	// We need to inspect the error chain to find a potential gRPC status error,
	// as it might be wrapped by other errors (e.g., using fmt.Errorf).
	var grpcStatusError error
	u := w.err
	for u != nil {
		// Check if the error can provide a gRPC status.
		if _, ok := u.(interface{ GRPCStatus() *status.Status }); ok {
			// To avoid recursion with our own type, we skip errWithMetadata
			// and continue unwrapping. We are looking for the original gRPC status.
			if _, isOurType := u.(*errWithMetadata); !isOurType { // nolint: errorlint // errors.As should not be used here
				grpcStatusError = u
				break
			}
		}
		u = errors.Unwrap(u)
	}
	// Check which error to use to get the Status
	errToConvert := w.err
	if grpcStatusError != nil {
		errToConvert = grpcStatusError
	}
	baseStatus := status.Convert(errToConvert)
	// Collect all metadata from the entire error chain, starting from the current error.
	allMetadata := GetMetadata(w)
	// If there's no metadata, just return the status.
	if len(allMetadata) == 0 {
		return baseStatus
	}
	// Convert our metadata slice into a map for structpb.
	metadataMap := make(map[string]any)
	for i := 0; i < len(allMetadata); i += 2 {
		key, ok := allMetadata[i].(string)
		if !ok {
			// Keys must be strings for structpb.
			continue
		}
		if i+1 >= len(allMetadata) {
			break
		}
		metadataMap[key] = allMetadata[i+1]
	}
	// If we successfully converted some metadata, create a struct.
	if len(metadataMap) > 0 {
		metadataStruct, err := structpb.NewStruct(metadataMap)
		if err == nil {
			// Create a new status with the same code and message, but without the original details.
			st := status.New(baseStatus.Code(), baseStatus.Message())
			// Attach the struct as a detail to the status.
			if stWithDetails, err := st.WithDetails(metadataStruct); err == nil {
				return stWithDetails
			}
		}
	}
	// Fallback to returning the original status if metadata couldn't be attached.
	return baseStatus
}

// Unwrap returns the original error that was wrapped with errWithMetadata instance
// It makes the errWithMetadata compatible with the standard error unwrapping mechanism
func (w *errWithMetadata) Unwrap() error {
	return w.err
}

type Metadata []any

// Extend returns a new metadata container with combined key value pairs from current metadata and provided key value pairs
func (m *Metadata) Extend(keyValues ...any) Metadata {
	if m == nil {
		return keyValues
	}
	return mergeKeyValuePair(*m, keyValues)
}

// WithMetadata returns the provided error wrapped with the provided metadata
func WithMetadata(err error, keyValues ...any) error {
	if err == nil {
		return nil
	}
	// try to detect types of provided keyValues and build up proper key value pair
	metadata := make([]any, 0)
	for _, kv := range keyValues {
		t := reflect.TypeOf(kv)
		switch t.Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(kv)
			// We need to use .Interface() to get the actual value, not the reflect.Value
			for i := 0; i < s.Len(); i++ {
				metadata = append(metadata, s.Index(i).Interface())
			}
		case reflect.Map:
			// Use reflection to iterate over the map to handle any map type
			// without panicking on type assertion.
			v := reflect.ValueOf(kv)
			iter := v.MapRange()
			for iter.Next() {
				metadata = append(metadata, iter.Key().Interface(), iter.Value().Interface())
			}
		default:
			metadata = append(metadata, kv)
		}
	}

	return &errWithMetadata{
		err:      err,
		metadata: metadata,
	}
}

// GetMetadata returns metadata from the error chain
// If there is no metadata in the chain, it will return an empty slice
// It returns []any to make it compatible with structured logging libraries (like slog, zap, or logr).
func GetMetadata(err error) []any {
	if err == nil {
		return []any{}
	}

	// Recursively get metadata from the wrapped error first. This ensures that
	// metadata from the innermost error is collected first.
	metadata := GetMetadata(errors.Unwrap(err))

	// Then, append metadata from the current error level. This way, when the
	// resulting slice is converted to a map, keys from outer (more recent)
	// wrappers will overwrite keys from inner wrappers, giving them precedence.
	// This is compatible with the "last one wins" behavior of most structured loggers.
	if e, ok := err.(*errWithMetadata); ok { // nolint: errorlint
		metadata = append(metadata, e.metadata...)
	} else {
		// This captures metadata from errors that conform to the gRPC status interface.
		if s, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
			st := s.GRPCStatus()
			for _, detail := range st.Details() {
				if metadataStruct, ok := detail.(*structpb.Struct); ok {
					for key, val := range metadataStruct.GetFields() {
						metadata = append(metadata, key, val.AsInterface())
					}
				}
			}
		}
	}
	return metadata
}

// mergeKeyValuePair merges two slices into a new slice.
// It assumes that both slices are valid key value pairs.
// If a key is missing a value, it will add a padding "<missing>" to the slice.
func mergeKeyValuePair(cur, new []any) []any {
	// Both "cur" and "new" should be valid key value pair.
	// We will be adding a padding in case some key misses value.
	paddedCur := addPaddingForMissingValue(cur)
	paddedNew := addPaddingForMissingValue(new)
	// just to avoid reallocation, we will create a new slice with the combined length
	newKV := make([]any, 0, len(paddedCur)+len(paddedNew))
	newKV = append(newKV, paddedCur...)
	newKV = append(newKV, paddedNew...)
	return newKV
}

// addPaddingForMissingValue adds a padding "<missing>" to the slice if the last key is missing a value
func addPaddingForMissingValue(keyValues []any) []any {
	newLen := len(keyValues)
	// check if the last key has a value
	missingValue := len(keyValues)%2 != 0
	if missingValue {
		newLen++
	}

	// create a new slice with the new length
	newKV := make([]any, 0, newLen)
	// copy the key values to the new slice
	newKV = append(newKV, keyValues...)
	// add padding if the last key is missing a value
	if missingValue {
		newKV = append(newKV, "<missing>")
	}
	return newKV
}
