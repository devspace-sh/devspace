package registry

import (
	"gotest.tools/assert"
	"testing"
)

func TestRegistry(t *testing.T) {
	newRegistry := NewDependencyRegistry("test", true)
	locked, err := newRegistry.TryLockDependencies(nil, "test", []string{"test"}, true)
	assert.NilError(t, err)
	assert.DeepEqual(t, locked, map[string]LockType{
		"test": Locked,
	})

	locked, err = newRegistry.TryLockDependencies(nil, "test", []string{"dep1", "dep2", "dep3"}, false)
	assert.NilError(t, err)
	assert.DeepEqual(t, locked, map[string]LockType{
		"dep1": Locked,
		"dep2": Locked,
		"dep3": Locked,
	})

	locked, err = newRegistry.TryLockDependencies(nil, "dep1", []string{"test", "dep2"}, false)
	assert.NilError(t, err)
	assert.DeepEqual(t, locked, map[string]LockType{
		"test": InUseCyclic,
		"dep2": InUse,
	})

	locked, err = newRegistry.TryLockDependencies(nil, "dep1", []string{"dep1"}, false)
	assert.NilError(t, err)
	assert.DeepEqual(t, locked, map[string]LockType{
		"dep1": InUseCyclic,
	})

	locked, err = newRegistry.TryLockDependencies(nil, "dep2", []string{"dep1"}, false)
	assert.NilError(t, err)
	assert.DeepEqual(t, locked, map[string]LockType{
		"dep1": InUseCyclic,
	})
}
