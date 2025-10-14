package api

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReadOnlyInterceptor creates a gRPC unary interceptor that only allows read-only operations.
// This is used for the Unix socket listener to prevent write operations from local CLI.
func ReadOnlyInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if this is a read-only method
		if !isReadOnlyMethod(info.FullMethod) {
			return nil, status.Errorf(
				codes.PermissionDenied,
				"write operations not allowed on Unix socket - use TCP connection with mTLS (warren init --manager <addr> --token <token>)",
			)
		}

		// Allow read-only operations
		return handler(ctx, req)
	}
}

// isReadOnlyMethod checks if a gRPC method is read-only
func isReadOnlyMethod(method string) bool {
	// Extract method name from full path (e.g., "/proto.WarrenAPI/ListServices" -> "ListServices")
	parts := strings.Split(method, "/")
	if len(parts) < 2 {
		return false
	}
	methodName := parts[len(parts)-1]

	// Read-only methods (List*, Get*, Inspect*, Watch*, Stream* for reads)
	readOnlyPrefixes := []string{
		"List",
		"Get",
		"Inspect",
		"Watch",
		"Describe",
		"Show",
	}

	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(methodName, prefix) {
			return true
		}
	}

	// Special cases: StreamEvents is read-only (event streaming)
	readOnlyMethods := []string{
		"StreamEvents",
		"GetClusterInfo",
		"GetNodeInfo",
		"GetServiceInfo",
	}

	for _, allowedMethod := range readOnlyMethods {
		if methodName == allowedMethod {
			return true
		}
	}

	// Default: block
	return false
}
