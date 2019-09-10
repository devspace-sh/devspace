package hook

import (
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func TestHookWithoutExecution(t *testing.T) {
	//Execute 0 hooks
	err := Execute(&latest.Config{}, 0, 0, "", nil)
	if err != nil {
		t.Fatalf("Failed to execute 0 hooks with error: %v", err)
	}

	//Execute 1 hook with no when
	err = Execute(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{},
		},
	}, 0, 0, "", nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without when with error: %v", err)
	}

	//Execute 1 hook with no When.Before and no When.After
	err = Execute(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{},
			},
		},
	}, 0, 0, "", nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without When.Before and When.After with error: %v", err)
	}

	//Execute 1 hook with empty When.Before
	err = Execute(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					Before: &latest.HookWhenAtConfig{},
				},
			},
		},
	}, Before, 0, "", nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.Before: %v", err)
	}

	//Execute 1 hook with empty When.After
	err = Execute(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					After: &latest.HookWhenAtConfig{},
				},
			},
		},
	}, After, 0, "", nil)
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}

}

func TestHookWithExecution(t *testing.T){
	err := Execute(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					Before: &latest.HookWhenAtConfig{
						Deployments: "theseDeployments",
					},
				},
				Command: "echo",
				Args: []string{"hello"},
			},
		},
	}, Before, StageDeployments, "theseDeployments", &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}

}
