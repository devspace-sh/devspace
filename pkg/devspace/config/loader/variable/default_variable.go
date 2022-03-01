package variable

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"os"
	"strconv"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
)

// NewDefaultVariable creates a new variable for the sources default, all or input
func NewDefaultVariable(name string, workingDirectory string, localCache localcache.Cache, remoteCache remotecache.Cache, log log.Logger) Variable {
	return &defaultVariable{
		name:             name,
		workingDirectory: workingDirectory,
		localCache:       localCache,
		remoteCache:      remoteCache,
		log:              log,
	}
}

type defaultVariable struct {
	name             string
	workingDirectory string
	localCache       localcache.Cache
	remoteCache      remotecache.Cache
	log              log.Logger
}

func (d *defaultVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	if definition.Command != "" || len(definition.Commands) > 0 {
		return NewCommandVariable(d.name, d.workingDirectory).Load(ctx, definition)
	}

	// Check environment
	value := os.Getenv(d.name)

	// Did we find it in the environment variables?
	if definition.Source != latest.VariableSourceInput && value != "" {
		return valueByType(value, definition.Default)
	}

	// Remote cache takes precedence over local cache
	if !definition.NoCache {
		if definition.RemoteCache {
			if value, ok := d.remoteCache.GetVar(d.name); !definition.NoCache && ok {
				return valueByType(value, definition.Default)
			}
		} else {
			if value, ok := d.localCache.GetVar(d.name); !definition.NoCache && ok {
				return valueByType(value, definition.Default)
			}
		}
	}

	// Now ask the question
	value, err := askQuestion(definition, d.log)
	if err != nil {
		return nil, err
	}

	if !definition.NoCache {
		if definition.RemoteCache {
			d.remoteCache.SetVar(d.name, value)
		} else {
			d.localCache.SetVar(d.name, value)
		}
	}
	return valueByType(value, definition.Default)
}

func valueByType(value string, defaultValue interface{}) (interface{}, error) {
	if defaultValue == nil {
		return convertStringValue(value), nil
	}

	switch defaultValue.(type) {
	case int:
		r, err := strconv.Atoi(value)
		return r, err
	case bool:
		r, err := strconv.ParseBool(value)
		return r, err
	default:
		return value, nil
	}
}
