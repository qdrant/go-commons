package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		{
			name:     "gRPC status error",
			err:      status.Error(codes.NotFound, "item not found"),
			expected: "rpc error: code = NotFound desc = item not found",
		},
		{
			name:     "wrapped gRPC status error",
			err:      WithMetadata(status.Error(codes.NotFound, "item not found"), "key", "value"),
			expected: "rpc error: code = NotFound desc = item not found",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestGRPCStatus(t *testing.T) {
	plainErr := errors.New("plain error")
	grpcErr := status.Error(codes.NotFound, "item not found")
	expectedGrpcStatus, ok := status.FromError(grpcErr)
	require.True(t, ok)

	testCases := []struct {
		name            string
		err             error
		expectedMessage string
		expectedStatus  *status.Status
		expectOk        bool
	}{
		{
			name:            "nil error",
			err:             nil,
			expectedMessage: "",  // A nil error is converted to an OK status, which has an empty message.
			expectedStatus:  nil, // status.FromError(nil) returns (nil, true), which is treated as OK.
			expectOk:        true,
		},
		{
			name:            "standard error",
			err:             plainErr,
			expectedMessage: "plain error",
			expectedStatus:  status.New(codes.Unknown, "plain error"),
			expectOk:        false,
		},
		{
			name:            "gRPC status error",
			err:             grpcErr,
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatus,
			expectOk:        true,
		},
		{
			name:            "standard error wrapped with metadata",
			err:             WithMetadata(plainErr, "key", "value"),
			expectedMessage: "plain error",
			expectedStatus:  status.New(codes.Unknown, "plain error"),
			expectOk:        false,
		},
		{
			name:            "gRPC status error wrapped with metadata",
			err:             WithMetadata(grpcErr, "key", "value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatus,
			expectOk:        true,
		},
		{
			name:            "gRPC status error wrapped with fmt.Errorf then metadata",
			err:             WithMetadata(fmt.Errorf("wrapped: %w", grpcErr), "key", "value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatus,
			expectOk:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			st, ok := status.FromError(tc.err)
			require.Equal(t, tc.expectOk, ok)
			require.Equal(t, tc.expectedMessage, status.Convert(tc.err).Message())
			if tc.expectedStatus == nil {
				require.Nil(t, st)
			} else {
				require.NotNil(t, st)
				require.Equal(t, tc.expectedStatus.Proto(), st.Proto())
			}
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
