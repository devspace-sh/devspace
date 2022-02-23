package hook

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func TestHookWithoutExecution(t *testing.T) {
	// Execute 0 hooks
	conf := config.NewConfig(nil, &latest.Config{}, nil, nil, constants.DefaultConfigPath)
	err := ExecuteHooks(devspacecontext.NewContext().WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 0 hooks with error: %v", err)
	}

	// Execute 1 hook with no when
	conf = config.NewConfig(nil, &latest.Config{
		Hooks: []*latest.HookConfig{
			{},
		},
	}, nil, nil, constants.DefaultConfigPath)
	err = ExecuteHooks(devspacecontext.NewContext().WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without when with error: %v", err)
	}

	// Execute 1 hook with no When.Before and no When.After
	conf = config.NewConfig(nil, &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events: []string{"before:deploy"},
			},
		},
	}, nil, nil, constants.DefaultConfigPath)
	err = ExecuteHooks(devspacecontext.NewContext().WithConfig(conf), nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without When.Before and When.After with error: %v", err)
	}
}

func TestHookWithExecution(t *testing.T) {
	conf := config.NewConfig(nil, &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events:  []string{"my-event"},
				Command: "echo",
				Args:    []string{"hello"},
			},
		},
	}, nil, nil, constants.DefaultConfigPath)
	err := ExecuteHooks(devspacecontext.NewContext().WithConfig(conf), nil, "my-event")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}
}
