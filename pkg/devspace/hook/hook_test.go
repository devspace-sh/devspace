package hook

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func TestHookWithoutExecution(t *testing.T) {
	//Execute 0 hooks
	executer := NewExecuter(&latest.Config{}, log.Discard)
	err := executer.Execute(0, 0, "")
	if err != nil {
		t.Fatalf("Failed to execute 0 hooks with error: %v", err)
	}

	//Execute 1 hook with no when
	executer = NewExecuter(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{},
		},
	}, log.Discard)
	err = executer.Execute(0, 0, "")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without when with error: %v", err)
	}

	//Execute 1 hook with no When.Before and no When.After
	executer = NewExecuter(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{},
			},
		},
	}, log.Discard)
	err = executer.Execute(0, 0, "")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook without When.Before and When.After with error: %v", err)
	}

	//Execute 1 hook with empty When.Before
	executer = NewExecuter(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					Before: &latest.HookWhenAtConfig{},
				},
			},
		},
	}, log.Discard)
	err = executer.Execute(Before, 0, "")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.Before: %v", err)
	}

	//Execute 1 hook with empty When.After
	executer = NewExecuter(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					After: &latest.HookWhenAtConfig{},
				},
			},
		},
	}, log.Discard)
	err = executer.Execute(After, 0, "")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}

}

func TestHookWithExecution(t *testing.T) {
	executer := NewExecuter(&latest.Config{
		Hooks: []*latest.HookConfig{
			&latest.HookConfig{
				When: &latest.HookWhenConfig{
					Before: &latest.HookWhenAtConfig{
						Deployments: "theseDeployments",
					},
				},
				Command: "echo",
				Args:    []string{"hello"},
			},
		},
	}, log.Discard)
	err := executer.Execute(Before, StageDeployments, "theseDeployments")
	if err != nil {
		t.Fatalf("Failed to execute 1 hook with empty When.After: %v", err)
	}

}
