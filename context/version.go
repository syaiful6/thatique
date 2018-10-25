package context

import "context"


type versionKey struct{}

func (versionKey) String() string { return "version" }

// WithVersion stores the application version in the context. The new context
// gets a logger to ensure log messages are marked with the application
// version.
func WithVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, versionKey{}, version)
}

// GetVersion returns the application version from the context. An empty
// string may returned if the version was not set on the context.
func GetVersion(ctx context.Context) string {
	return GetStringValue(ctx, versionKey{})
}