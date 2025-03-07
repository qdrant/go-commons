package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "error without metadata",
			err:      errors.New("foo"),
			expected: "foo",
		},
		{
			name:     "error with metadata",
			err:      WithMetadata(errors.New("foo"), "key", "value"),
			expected: "foo",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestErrWrapper_Extend(t *testing.T) {
	// create error context with some metadata
	errContext := Context("k1", "v1")
	// extend the error context with additional metadata
	extendedContext := errContext.Extend("k2", "v2")
	// verify that the extended context contains both original and new metadata
	require.Equal(t, []any{"k1", "v1", "k2", "v2"}, extendedContext.metadata)
}

func TestErrWrapper_With(t *testing.T) {
	testCases := []struct {
		name        string
		providedErr error
		wrapper     *errWrapper
		expectedErr error
	}{
		{
			name:        "when provided error is nil",
			providedErr: nil,
			wrapper:     &errWrapper{},
			expectedErr: nil,
		},
		{
			name:        "empty wrapper",
			providedErr: errors.New("foo"),
			wrapper:     &errWrapper{},
			expectedErr: &errWrapper{
				err: errors.New("foo"),
			},
		},
		{
			name:        "wrapper with metadata",
			providedErr: errors.New("foo"),
			wrapper: &errWrapper{
				metadata: []any{"key", "value"},
			},
			expectedErr: &errWrapper{
				err:      errors.New("foo"),
				metadata: []any{"key", "value"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualErr := tc.wrapper.With(tc.providedErr)
			require.Equal(t, tc.expectedErr, actualErr)
		})
	}
}

func TestErrWrapper_WithMetadata(t *testing.T) {
	fooError := errors.New("foo")
	barError := errors.New("bar")
	testCases := []struct {
		name     string
		current  *errWrapper
		metadata []any
		err      error
		expected *errWrapper
	}{
		{
			name:     "when wrapper is nil",
			current:  nil,
			err:      nil,
			metadata: []any{"key", "value"},
			expected: nil,
		},
		{
			name:     "when error is nil",
			current:  &errWrapper{},
			err:      nil,
			metadata: []any{"key", "value"},
			expected: nil,
		},
		{
			name:     "when current wrapper is empty",
			err:      fooError,
			current:  &errWrapper{},
			metadata: nil,
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{},
			},
		},
		{
			name: "when current wrapper has metadata",
			err:  fooError,
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: nil,
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "oldValue"},
			},
		},
		{
			name: "when provided metadata is empty",
			err:  fooError,
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "oldValue"},
			},
		},
		{
			name:     "when provided metadata is not empty and current wrapper is empty",
			err:      fooError,
			current:  &errWrapper{},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"newKey", "newValue"},
			},
		},
		{
			name: "when provided metadata is not empty and current wrapper has metadata",
			err:  fooError,
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "oldValue", "newKey", "newValue"},
			},
		},
		{
			name: "when current wrapper has error and metadata",
			err:  fooError,
			current: &errWrapper{
				err:      barError,
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "oldValue", "newKey", "newValue"},
			},
		},
		{
			name: "when current metadata misses a value",
			err:  fooError,
			current: &errWrapper{
				metadata: []any{"oldKey"},
			},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "<missing>", "newKey", "newValue"},
			},
		},
		{
			name: "when provided metadata misses a value",
			err:  fooError,
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{"newKey"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"oldKey", "oldValue", "newKey", "<missing>"},
			},
		},
		{
			name: "when provided error is already wrapped with metadata",
			err:  WithMetadata(fooError, "key", "value"),
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      WithMetadata(fooError, "key", "value"),
				metadata: []any{"oldKey", "oldValue", "newKey", "newValue"},
			},
		},
		{
			name: "when provided error is already wrapped with custom message",
			err:  fmt.Errorf("bar: %w", fooError),
			current: &errWrapper{
				metadata: []any{"oldKey", "oldValue"},
			},
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fmt.Errorf("bar: %w", fooError),
				metadata: []any{"oldKey", "oldValue", "newKey", "newValue"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.current.WithMetadata(tc.err, tc.metadata...)
			if tc.expected == nil {
				require.NoError(t, actual)
			} else {
				require.Error(t, actual)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestWithMetadata(t *testing.T) {
	fooError := errors.New("foo")
	testCases := []struct {
		name     string
		metadata []any
		err      error
		expected *errWrapper
	}{
		{
			name:     "when error is nil",
			err:      nil,
			metadata: []any{"key", "value"},
			expected: nil,
		},
		{
			name:     "when provided metadata is empty",
			err:      fooError,
			metadata: []any{},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{},
			},
		},
		{
			name:     "when provided metadata is not empty",
			err:      fooError,
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"newKey", "newValue"},
			},
		},
		{
			name:     "when provided metadata misses a value",
			err:      fooError,
			metadata: []any{"newKey"},
			expected: &errWrapper{
				err:      fooError,
				metadata: []any{"newKey", "<missing>"},
			},
		},
		{
			name:     "when provided error is already wrapped with metadata",
			err:      WithMetadata(fooError, "key", "value"),
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      WithMetadata(fooError, "key", "value"),
				metadata: []any{"newKey", "newValue"},
			},
		},
		{
			name:     "when provided error is already wrapped with custom message",
			err:      fmt.Errorf("bar: %w", fooError),
			metadata: []any{"newKey", "newValue"},
			expected: &errWrapper{
				err:      fmt.Errorf("bar: %w", fooError),
				metadata: []any{"newKey", "newValue"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := WithMetadata(tc.err, tc.metadata...)
			if tc.expected == nil {
				require.NoError(t, actual)
			} else {
				require.Error(t, actual)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	rootError := errors.New("this is root error")

	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "error without wrapping",
			err:      errors.New("plain error"),
			expected: nil,
		},
		{
			name:     "error wrapped with metadata",
			err:      WithMetadata(rootError, "key", "value"),
			expected: rootError,
		},
		{
			name:     "error wrapped with custom message",
			err:      fmt.Errorf("foo: %w", rootError),
			expected: rootError,
		},
		{
			name:     "error wrapped in multiple levels with metadata",
			err:      WithMetadata(WithMetadata(rootError, "k1", "v1"), "k2", "v2"),
			expected: WithMetadata(rootError, "k1", "v1"),
		},
		{
			name:     "error wrapped in multiple levels with custom message",
			err:      fmt.Errorf("foo: %w", fmt.Errorf("bar: %w", rootError)),
			expected: fmt.Errorf("bar: %w", rootError),
		},
		{
			name:     "error wrapped in multiple levels with metadata and custom message",
			err:      fmt.Errorf("foo: %w", WithMetadata(rootError, "k1", "v1")),
			expected: WithMetadata(rootError, "k1", "v1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := errors.Unwrap(tt.err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetMetadata(t *testing.T) {
	rootError := errors.New("this is root error")
	testCases := []struct {
		name     string
		err      error
		expected []any
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: []any{},
		},
		{
			name:     "error without wrapping",
			err:      rootError,
			expected: []any{},
		},
		{
			name:     "error wrapped without metadata",
			err:      WithMetadata(rootError),
			expected: []any{},
		},
		{
			name:     "error wrapped with metadata",
			err:      WithMetadata(rootError, "key", "value"),
			expected: []any{"key", "value"},
		},
		{
			name:     "error wrapped with custom message",
			err:      fmt.Errorf("foo: %w", rootError),
			expected: []any{},
		},
		{
			name:     "error wrapped in multiple levels with metadata",
			err:      WithMetadata(WithMetadata(rootError, "k1", "v1"), "k2", "v2"),
			expected: []any{"k2", "v2", "k1", "v1"},
		},
		{
			name:     "error wrapped in multiple levels with custom message",
			err:      fmt.Errorf("foo: %w", fmt.Errorf("bar: %w", rootError)),
			expected: []any{},
		},
		{
			name:     "error wrapped in multiple levels with metadata and custom message",
			err:      fmt.Errorf("foo: %w", WithMetadata(rootError, "k1", "v1")),
			expected: []any{"k1", "v1"},
		},
		{
			name:     "error wrapped with metadata is at the middle of the chain",
			err:      fmt.Errorf("foo: %w", WithMetadata(fmt.Errorf("bar: %w", rootError), "k1", "v1")),
			expected: []any{"k1", "v1"},
		},
		{
			name:     "error wrapped with metadata is at the beginning and end of the chain",
			err:      WithMetadata(fmt.Errorf("foo: %w", WithMetadata(rootError, "k1", "v1")), "k2", "v2"),
			expected: []any{"k2", "v2", "k1", "v1"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetMetadata(tc.err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsNonRetryableError(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		isNonRetryable bool
	}{
		{
			name:           "error without wrapping",
			err:            errors.New("foo"),
			isNonRetryable: false,
		},
		{
			name:           "error wrapped with message",
			err:            fmt.Errorf("foo: %w", errors.New("bar")),
			isNonRetryable: false,
		},
		{
			name:           "error wrapped with metadata",
			err:            WithMetadata(errors.New("foo"), "key", "value"),
			isNonRetryable: false,
		},
		{
			name:           "non-retryable error wrapped with metadata",
			err:            AsNonRetryableError(errors.New("foo"), "key", "value"),
			isNonRetryable: true,
		},
		{
			name:           "non-retryable error without wrapping",
			err:            &nonRetryableError{err: errors.New("foo")},
			isNonRetryable: true,
		},
		{
			name:           "non-retryable error in the error chain",
			err:            fmt.Errorf("foo: %w", &nonRetryableError{err: errors.New("bar")}),
			isNonRetryable: true,
		},
		{
			name:           "retryable error",
			err:            &retryableError{err: errors.New("foo")},
			isNonRetryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isNonRetryable, IsNonRetryableError(tc.err))
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		isRetryable bool
	}{
		{
			name:        "error without wrapping",
			err:         errors.New("foo"),
			isRetryable: false,
		},
		{
			name:        "error wrapped with message",
			err:         fmt.Errorf("foo: %w", errors.New("bar")),
			isRetryable: false,
		},
		{
			name:        "error wrapped with metadata",
			err:         WithMetadata(errors.New("foo"), "key", "value"),
			isRetryable: false,
		},
		{
			name:        "retryable error wrapped with metadata",
			err:         AsRetryableError(errors.New("foo"), "key", "value"),
			isRetryable: true,
		},
		{
			name:        "retryable error without wrapping",
			err:         &retryableError{err: errors.New("foo")},
			isRetryable: true,
		},
		{
			name:        "retryable error in the error chain",
			err:         fmt.Errorf("foo: %w", &retryableError{err: errors.New("bar")}),
			isRetryable: true,
		},
		{
			name:        "non-retryable error",
			err:         &nonRetryableError{err: errors.New("foo")},
			isRetryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isRetryable, IsRetryableError(tc.err))
		})
	}
}
