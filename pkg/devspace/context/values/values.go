package values

import (
	"context"
	flag "github.com/spf13/pflag"
	"strings"
)

// The key type is unexported to prevent collisions
type key int

const (
	// nameKey is the context key for the DevSpace name.
	nameKey key = iota
	tempFolderKey
	dependencyKey
	rootNameKey
	devContextKey
	flagsKey
	commandFlagsKey
)

// WithFlagsMap creates a new context with the given flags
func WithFlagsMap(parent context.Context, flagsMap map[string]string) context.Context {
	return WithValue(parent, flagsKey, flagsMap)
}

// WithFlags creates a new context with the given flags
func WithCommandFlags(parent context.Context, flagSet *flag.FlagSet) context.Context {
	flagsMap := map[string]string{}
	flagSet.VisitAll(func(f *flag.Flag) {
		sliceType, ok := f.Value.(flag.SliceValue)
		if ok {
			flagsMap[f.Name] = strings.Join(sliceType.GetSlice(), " ")
		} else {
			flagsMap[f.Name] = f.Value.String()
		}
	})

	gfc := WithValue(parent, commandFlagsKey, flagsMap)
	return WithValue(gfc, flagsKey, flagsMap)
}

// FlagsFrom returns a context used to start and stop dev configurations
func FlagsFrom(ctx context.Context) (map[string]string, bool) {
	flags, fOk := ctx.Value(flagsKey).(map[string]string)
	commandFlags, cOk := ctx.Value(commandFlagsKey).(map[string]string)
	return mergeFlags(commandFlags, flags), fOk && cOk
}

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

func WithDependency(parent context.Context, dependency bool) context.Context {
	return WithValue(parent, dependencyKey, dependency)
}

func IsDependencyFrom(ctx context.Context) (bool, bool) {
	isDependency, ok := ctx.Value(dependencyKey).(bool)
	return isDependency, ok
}

func mergeFlags(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}
