package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
	// Create expected status with details for the metadata test
	metadataStruct, err := structpb.NewStruct(map[string]any{"key": "value"})
	require.NoError(t, err)
	expectedGrpcStatusWithDetails, err := expectedGrpcStatus.WithDetails(metadataStruct)
	require.NoError(t, err)

	// Create expected status with details for the nested metadata test
	nestedMetadataMap := map[string]any{
		"outer_key": "outer_value",
		"inner_key": "inner_value",
	}
	nestedMetadataStruct, err := structpb.NewStruct(nestedMetadataMap)
	require.NoError(t, err)
	expectedGrpcStatusWithNestedDetails, err := expectedGrpcStatus.WithDetails(nestedMetadataStruct)
	require.NoError(t, err)

	// Create expected status for reused key test
	reusedKeyMap := map[string]any{
		"reused_key": "outer_value", // The outer value should win
	}
	reusedKeyStruct, err := structpb.NewStruct(reusedKeyMap)
	require.NoError(t, err)
	expectedStatusWithReusedKey, err := expectedGrpcStatus.WithDetails(reusedKeyStruct)
	require.NoError(t, err)

	// Create an error that simulates one received from another service, with its own metadata
	stRemote := status.New(codes.Aborted, "remote operation failed")
	remoteMetaStruct, err := structpb.NewStruct(map[string]any{
		"remote_key": "remote_value",
		"shared_key": "remote_shared_value",
	})
	require.NoError(t, err)
	stRemoteWithDetails, err := stRemote.WithDetails(remoteMetaStruct)
	require.NoError(t, err)
	remoteErrWithDetails := stRemoteWithDetails.Err()

	// Create the expected final status for the chaining test
	// The final map will have local keys and the remote key, with the local shared_key overwriting the remote one.
	finalCombinedMap := map[string]any{
		"remote_key": "remote_value",
		"shared_key": "local_shared_value", // This one overwrites the remote one
		"local_key":  "local_value",
	}
	finalCombinedStruct, err := structpb.NewStruct(finalCombinedMap)
	require.NoError(t, err)
	expectedFinalStatus, err := stRemote.WithDetails(finalCombinedStruct)
	require.NoError(t, err)

	expectedUnknownStatusWithDetails, err := status.New(codes.Unknown, "plain error").WithDetails(metadataStruct)
	require.NoError(t, err)

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
			expectedStatus:  expectedUnknownStatusWithDetails,
			expectOk:        true,
		},
		{
			name:            "gRPC status error wrapped with metadata",
			err:             WithMetadata(grpcErr, "key", "value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatusWithDetails,
			expectOk:        true,
		},
		{
			name:            "gRPC status error wrapped with fmt.Errorf then metadata",
			err:             WithMetadata(fmt.Errorf("wrapped: %w", grpcErr), "key", "value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatusWithDetails,
			expectOk:        true,
		},
		{
			name:            "gRPC status error wrapped with nested metadata",
			err:             WithMetadata(WithMetadata(grpcErr, "inner_key", "inner_value"), "outer_key", "outer_value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedGrpcStatusWithNestedDetails,
			expectOk:        true,
		},
		{
			name:            "gRPC status error wrapped with reused metadata key",
			err:             WithMetadata(WithMetadata(grpcErr, "reused_key", "inner_value"), "reused_key", "outer_value"),
			expectedMessage: "item not found",
			expectedStatus:  expectedStatusWithReusedKey,
			expectOk:        true,
		},
		{
			name:            "error with gRPC details wrapped with more metadata with overlapping keys",
			err:             WithMetadata(remoteErrWithDetails, "local_key", "local_value", "shared_key", "local_shared_value"),
			expectedMessage: "remote operation failed",
			expectedStatus:  expectedFinalStatus,
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
				// Comparing protobufs directly can be flaky if they contain maps, as the serialization order of map keys is not guaranteed.
				// Instead, we perform a semantic comparison of the status components.
				require.Equal(t, tc.expectedStatus.Code(), st.Code(), "gRPC status codes should be equal")
				require.Equal(t, tc.expectedStatus.Message(), st.Message(), "gRPC status messages should be equal")

				expectedDetails := tc.expectedStatus.Details()
				actualDetails := st.Details()
				require.Len(t, actualDetails, len(expectedDetails), "number of details should match")

				if len(expectedDetails) > 0 {
					// The Details() method returns the unmarshalled proto messages. We expect a *structpb.Struct.
					// We type-assert the detail and then compare the underlying maps.
					expectedStruct, ok := expectedDetails[0].(*structpb.Struct)
					require.True(t, ok, "expected detail should be a *structpb.Struct")
					actualStruct, ok := actualDetails[0].(*structpb.Struct)
					require.True(t, ok, "actual detail should be a *structpb.Struct")
					require.Equal(t, expectedStruct.AsMap(), actualStruct.AsMap(), "metadata maps should be equal")
				}
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

	// Create a gRPC status with metadata in details to simulate an error from a gRPC call
	st := status.New(codes.Internal, "internal error")
	metadataStruct, err := structpb.NewStruct(map[string]any{"grpc_key": "grpc_value"})
	require.NoError(t, err)
	stWithDetails, err := st.WithDetails(metadataStruct)
	require.NoError(t, err)
	grpcErrorWithDetails := stWithDetails.Err()

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
			expected: []any{"k1", "v1", "k2", "v2"},
		},
		{
			name: "error wrapped with reused key in metadata",
			err:  WithMetadata(WithMetadata(rootError, "reused_key", "inner_value"), "reused_key", "outer_value"),
			// The slice contains both pairs. When passed to a logger that uses the last value for a given key,
			// "outer_value" will be the one that is logged, which is the desired behavior.
			expected: []any{"reused_key", "inner_value", "reused_key", "outer_value"},
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
			expected: []any{"k1", "v1", "k2", "v2"},
		},
		{
			name:     "error with metadata in gRPC status details",
			err:      grpcErrorWithDetails,
			expected: []any{"grpc_key", "grpc_value"},
		},
		{
			name:     "error wrapped with metadata and has gRPC status details",
			err:      WithMetadata(grpcErrorWithDetails, "wrapper_key", "wrapper_value"),
			expected: []any{"grpc_key", "grpc_value", "wrapper_key", "wrapper_value"},
		},
		{
			name:     "chained error with local and gRPC metadata with overlapping keys",
			err:      WithMetadata(grpcErrorWithDetails, "local_key", "local_value", "shared_key", "local_shared_value"),
			expected: []any{"grpc_key", "grpc_value", "local_key", "local_value", "shared_key", "local_shared_value"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetMetadata(tc.err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
