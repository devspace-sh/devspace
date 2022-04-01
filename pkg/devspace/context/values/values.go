package values

import "context"

// The key type is unexported to prevent collisions
type key int

const (
	// nameKey is the context key for the DevSpace name.
	nameKey key = iota
	tempFolderKey
	commandKey
	dependencyKey
	rootNameKey
	devContextKey
)

// WithDevContext creates a new context with the dev context
func WithDevContext(parent context.Context, devCtx context.Context) context.Context {
	return WithValue(parent, devContextKey, devCtx)
}

// DevContextFrom returns a context used to start and stop dev configurations
func DevContextFrom(ctx context.Context) (context.Context, bool) {
	devCtx, ok := ctx.Value(devContextKey).(context.Context)
	return devCtx, ok
}

// RootNameFrom returns the root name of the devspace config
func RootNameFrom(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(rootNameKey).(string)
	return user, ok
}

// WithRootName returns a copy of parent with the root name included
func WithRootName(parent context.Context, name string) context.Context {
	return WithValue(parent, rootNameKey, name)
}

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// WithName returns a copy of parent in which the devspace name value is set
func WithName(parent context.Context, name string) context.Context {
	return WithValue(parent, nameKey, name)
}

// NameFrom returns the name of the devspace config
func NameFrom(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(nameKey).(string)
	return user, ok
}

// WithTempFolder returns a copy of parent in which the devspace temp folder is set
func WithTempFolder(parent context.Context, name string) context.Context {
	return WithValue(parent, tempFolderKey, name)
}

// TempFolderFrom returns the name of the temporary devspace folder
func TempFolderFrom(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(tempFolderKey).(string)
	return user, ok
}

// WithCommand returns a copy of parent in which the devspace command is set
func WithCommand(parent context.Context, name string) context.Context {
	return WithValue(parent, commandKey, name)
}

// CommandFrom returns the name of the devspace command
func CommandFrom(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(commandKey).(string)
	return user, ok
}

func WithDependency(parent context.Context, dependency bool) context.Context {
	return WithValue(parent, dependencyKey, dependency)
}

func IsDependencyFrom(ctx context.Context) (bool, bool) {
	isDependency, ok := ctx.Value(dependencyKey).(bool)
	return isDependency, ok
}
