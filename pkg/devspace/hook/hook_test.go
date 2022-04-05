package hook

import (
	"context"
	"testing"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func TestHookWithoutExecution(t *testing.T) {
	// Execute 0 hooks
	// conf := config.NewConfig(nil, &latest.Config{}, nil, nil, constants.DefaultConfigPath)
	conf := config.NewConfig(map[string]interface{}{},
		map[string]interface{}{},
		latest.NewRaw(),
		localcache.New(constants.DefaultCacheFolder),
		&remotecache.RemoteCache{},
		map[string]interface{}{},
		constants.DefaultConfigPath)
	err := ExecuteHooks(devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 0 hooks with error: %v", err)
	}

	conf = config.NewConfig(map[string]interface{}{},
		map[string]interface{}{},
		&latest.Config{
			Hooks: []*latest.HookConfig{{}},
		},
		localcache.New(constants.DefaultCacheFolder),
		&remotecache.RemoteCache{},
		map[string]interface{}{},
		constants.DefaultConfigPath)
	err = ExecuteHooks(devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without when with error: %v", err)
	}

	conf = config.NewConfig(map[string]interface{}{},
		map[string]interface{}{},
		&latest.Config{
			Hooks: []*latest.HookConfig{{
				Events: []string{"before:deploy"},
			}},
		},
		localcache.New(constants.DefaultCacheFolder),
		&remotecache.RemoteCache{},
		map[string]interface{}{},
		constants.DefaultConfigPath)

	err = ExecuteHooks(devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without When.Before and When.After with error: %v", err)
	}
}

func TestHookWithExecution(t *testing.T) {
	conf := config.NewConfig(map[string]interface{}{},
		map[string]interface{}{},
		&latest.Config{
			Hooks: []*latest.HookConfig{{
				Events:  []string{"my-event"},
				Command: "echo",
				Args:    []string{"hello"},
			}},
		},
		localcache.New(constants.DefaultCacheFolder),
		&remotecache.RemoteCache{},
		map[string]interface{}{},
		constants.DefaultConfigPath)

	err := ExecuteHooks(devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}
}
