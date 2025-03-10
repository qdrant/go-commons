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
			name:     "error without wrapping",
			err:      errors.New("foo"),
			expected: "foo",
		},
		{
			name:     "error with wrapped with metadata",
			err:      WithMetadata(errors.New("foo"), "key", "value"),
			expected: "foo",
		},
		{
			name:     "error with wrapped with custom message",
			err:      fmt.Errorf("bar: %w", errors.New("foo")),
			expected: "bar: foo",
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
	errMeta := Metadata{"k1", "v1"}
	// extend the error context with additional metadata
	extendedMetadata := errMeta.Extend("k2", "v2")
	// verify that the extended context contains both original and new metadata
	require.EqualValues(t, []any{"k1", "v1", "k2", "v2"}, extendedMetadata)
}

func TestWithMetadata(t *testing.T) {
	fooError := errors.New("foo")
	testCases := []struct {
		name        string
		curMetadata Metadata
		newMetadata []any
		err         error
		expected    *errWithMetadata
	}{
		{
			name:        "when error is nil",
			err:         nil,
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{"k2", "v2"},
			expected:    nil,
		},
		{
			name:        "when current metadata is nil",
			err:         fooError,
			curMetadata: nil,
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k2", "v2"},
			},
		},
		{
			name:        "when current metadata is empty",
			err:         fooError,
			curMetadata: Metadata{},
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k2", "v2"},
			},
		},
		{
			name:        "when new metadata is nil",
			err:         fooError,
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: nil,
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "v1"},
			},
		},
		{
			name:        "when new metadata is empty",
			err:         fooError,
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "v1"},
			},
		},
		{
			name:        "when both current and new metadata are nil",
			err:         fooError,
			curMetadata: nil,
			newMetadata: nil,
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{},
			},
		},
		{
			name:        "when both current and new metadata are not empty",
			err:         fooError,
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "v1", "k2", "v2"},
			},
		},
		{
			name:        "when current metadata misses a value",
			err:         fooError,
			curMetadata: Metadata{"k1"},
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "<missing>", "k2", "v2"},
			},
		},
		{
			name:        "when new metadata misses a value",
			err:         fooError,
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{"k2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "v1", "k2", "<missing>"},
			},
		},
		{
			name:        "when both current and new metadata misses a value",
			err:         fooError,
			curMetadata: Metadata{"k1"},
			newMetadata: []any{"k2"},
			expected: &errWithMetadata{
				err:      fooError,
				metadata: []any{"k1", "<missing>", "k2", "<missing>"},
			},
		},
		{
			name:        "when provided error is already wrapped with metadata",
			err:         WithMetadata(fooError, "k0", "v0"),
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      WithMetadata(fooError, "k0", "v0"),
				metadata: []any{"k1", "v1", "k2", "v2"},
			},
		},
		{
			name:        "when provided error is already wrapped with custom message",
			err:         fmt.Errorf("bar: %w", fooError),
			curMetadata: Metadata{"k1", "v1"},
			newMetadata: []any{"k2", "v2"},
			expected: &errWithMetadata{
				err:      fmt.Errorf("bar: %w", fooError),
				metadata: []any{"k1", "v1", "k2", "v2"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := WithMetadata(tc.err, tc.curMetadata.Extend(tc.newMetadata...)...)
			if tc.expected == nil {
				require.NoError(t, actual)
			} else {
				require.Error(t, actual)
				require.EqualValues(t, tc.expected, actual)
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
