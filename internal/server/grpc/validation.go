package grpc

import (
	"net/url"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validation helpers for gRPC service methods.
// These reduce boilerplate and ensure consistent error messages.

// RequiredString validates that a string field is not empty.
// Returns nil if valid, or an InvalidArgument error with the field name.
func RequiredString(value, fieldName string) error {
	if value == "" {
		return status.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	return nil
}

// RequiredURL validates that a URL string is not empty and is a valid URL.
// Returns nil if valid, or an InvalidArgument error.
func RequiredURL(value string) error {
	if value == "" {
		return status.Error(codes.InvalidArgument, "url is required")
	}

	if _, err := url.Parse(value); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid url: %v", err)
	}

	return nil
}

// RequiredPath validates that a path string is not empty.
// Returns nil if valid, or an InvalidArgument error.
func RequiredPath(value string) error {
	if value == "" {
		return status.Error(codes.InvalidArgument, "path is required")
	}

	return nil
}

// RequiredName validates that a name string is not empty.
// Returns nil if valid, or an InvalidArgument error.
func RequiredName(value string) error {
	if value == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}

	return nil
}

// RequiredOneOf validates that at least one of the provided values is not empty.
// Returns nil if at least one is valid, or an InvalidArgument error listing the field names.
func RequiredOneOf(fields map[string]string) error {
	for _, value := range fields {
		if value != "" {
			return nil
		}
	}

	// Build field names list
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}

	if len(names) == 2 {
		return status.Errorf(codes.InvalidArgument, "%s or %s is required", names[0], names[1])
	}

	return status.Error(codes.InvalidArgument, "at least one field is required")
}

// NotNil validates that a pointer/interface is not nil.
// Returns nil if valid, or an InvalidArgument error with the field name.
func NotNil(value any, fieldName string) error {
	if value == nil {
		return status.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}

	return nil
}

// CanceledError returns a standard request canceled error.
func CanceledError() error {
	return status.Error(codes.Canceled, "request canceled")
}

// NotFoundError returns a standard not found error with the resource type.
func NotFoundError(resourceType string) error {
	return status.Errorf(codes.NotFound, "%s not found", resourceType)
}

// InternalError returns an internal error with the given message.
func InternalError(msg string) error {
	return status.Error(codes.Internal, msg)
}

// InternalErrorf returns an internal error with formatted message.
func InternalErrorf(format string, args ...any) error {
	return status.Errorf(codes.Internal, format, args...)
}
