package loader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gotest.tools/assert"
)

func TestResolveImportsDownloadsSiblingImportsConcurrently(t *testing.T) {
	oldDependencyFolderPath := dependencyutil.DependencyFolderPath
	dependencyutil.DependencyFolderPath = filepath.Join(t.TempDir(), "dependencies")
	defer func() {
		dependencyutil.DependencyFolderPath = oldDependencyFolderPath
	}()

	const importCount int32 = 3
	var activeRequests int32
	var maxActiveRequests int32
	allInFlight := make(chan struct{})
	var closeAllInFlight sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&activeRequests, 1)
		defer atomic.AddInt32(&activeRequests, -1)

		for {
			maxActive := atomic.LoadInt32(&maxActiveRequests)
			if current <= maxActive || atomic.CompareAndSwapInt32(&maxActiveRequests, maxActive, current) {
				break
			}
		}

		if current == importCount {
			closeAllInFlight.Do(func() {
				close(allInFlight)
			})
		}

		select {
		case <-allInFlight:
		case <-r.Context().Done():
			return
		}

		name := strings.TrimPrefix(r.URL.Path, "/")
		_, _ = fmt.Fprintf(w, "version: %s\npipelines:\n  pipeline-%s:\n    run: echo %s\n", latest.Version, name, name)
	}))
	defer server.Close()

	resolver := newImportTestResolver(t)
	rawConfig := map[string]interface{}{
		"version": latest.Version,
		"name":    "test",
		"imports": []interface{}{
			map[string]interface{}{"path": server.URL + "/a"},
			map[string]interface{}{"path": server.URL + "/b"},
			map[string]interface{}{"path": server.URL + "/c"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		resolved map[string]interface{}
		err      error
	}
	results := make(chan result, 1)
	go func() {
		resolved, err := ResolveImports(ctx, resolver, t.TempDir(), rawConfig, log.Discard)
		results <- result{resolved: resolved, err: err}
	}()

	select {
	case <-allInFlight:
	case result := <-results:
		assert.NilError(t, result.err)
		t.Fatalf("expected all import downloads to be in flight before resolution completed")
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatalf("timed out waiting for concurrent import downloads")
	}

	var resolved map[string]interface{}
	select {
	case result := <-results:
		assert.NilError(t, result.err)
		resolved = result.resolved
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatalf("timed out waiting for import resolution")
	}

	maxActive := atomic.LoadInt32(&maxActiveRequests)
	assert.Assert(t, maxActive == 3, "expected 3 concurrent import downloads, got %d", maxActive)

	pipelines := resolved["pipelines"].(map[string]interface{})
	for _, name := range []string{"a", "b", "c"} {
		assert.Assert(t, pipelines["pipeline-"+name] != nil, "expected pipeline-%s to be imported", name)
	}
}

func TestResolveImportsKeepsOrderedMergeSemantics(t *testing.T) {
	tempDir := t.TempDir()
	writeImportFile(t, tempDir, "import-a.yaml", `version: v2beta1
pipelines:
  shared:
    run: echo import-a
  import-a:
    run: echo import-a
hooks:
  - command: echo import-a
`)
	writeImportFile(t, tempDir, "import-b.yaml", `version: v2beta1
pipelines:
  shared:
    run: echo import-b
  import-b:
    run: echo import-b
hooks:
  - command: echo import-b
`)

	resolver := newImportTestResolver(t)
	rawConfig := map[string]interface{}{
		"version": latest.Version,
		"name":    "test",
		"imports": []interface{}{
			map[string]interface{}{"path": "import-a.yaml"},
			map[string]interface{}{"path": "import-b.yaml"},
		},
		"pipelines": map[string]interface{}{
			"root": map[string]interface{}{
				"run": "echo root",
			},
		},
		"hooks": []interface{}{
			map[string]interface{}{"command": "echo root"},
		},
	}

	resolved, err := ResolveImports(context.Background(), resolver, tempDir, rawConfig, log.Discard)
	assert.NilError(t, err)

	pipelines := resolved["pipelines"].(map[string]interface{})
	assert.Equal(t, pipelines["root"].(map[string]interface{})["run"], "echo root")
	assert.Equal(t, pipelines["shared"].(map[string]interface{})["run"], "echo import-a")
	assert.Equal(t, pipelines["import-a"].(map[string]interface{})["run"], "echo import-a")
	assert.Equal(t, pipelines["import-b"].(map[string]interface{})["run"], "echo import-b")

	hooks := resolved["hooks"].([]interface{})
	assert.Equal(t, len(hooks), 3)
	assert.Equal(t, hooks[0].(map[string]interface{})["command"], "echo root")
	assert.Equal(t, hooks[1].(map[string]interface{})["command"], "echo import-a")
	assert.Equal(t, hooks[2].(map[string]interface{})["command"], "echo import-b")

	_, ok := resolved["imports"]
	assert.Assert(t, !ok, "expected imports to be removed after resolution")
}

func TestResolveImportsMergesAndReloadsImportedVariables(t *testing.T) {
	tempDir := t.TempDir()
	writeImportFile(t, tempDir, "import-a.yaml", `version: v2beta1
vars:
  IMPORT_A: import-a
`)
	writeImportFile(t, tempDir, "import-b.yaml", `version: v2beta1
vars:
  IMPORT_B: import-b
`)

	resolver := newImportTestResolver(t)
	rawConfig := map[string]interface{}{
		"version": latest.Version,
		"name":    "test",
		"imports": []interface{}{
			map[string]interface{}{"path": "import-a.yaml"},
			map[string]interface{}{"path": "import-b.yaml"},
		},
	}

	resolved, err := ResolveImports(context.Background(), resolver, tempDir, rawConfig, log.Discard)
	assert.NilError(t, err)

	vars := resolved["vars"].(map[string]interface{})
	assert.Assert(t, vars["IMPORT_A"] != nil, "expected IMPORT_A to be present in merged vars")
	assert.Assert(t, vars["IMPORT_B"] != nil, "expected IMPORT_B to be present in merged vars")
	assert.Assert(t, resolver.DefinedVars()["IMPORT_A"] != nil, "expected resolver to know IMPORT_A")
	assert.Assert(t, resolver.DefinedVars()["IMPORT_B"] != nil, "expected resolver to know IMPORT_B")
}

func TestResolveImportsSkipsDisabledImports(t *testing.T) {
	tempDir := t.TempDir()
	writeImportFile(t, tempDir, "enabled.yaml", `version: v2beta1
pipelines:
  enabled:
    run: echo enabled
`)

	resolver := newImportTestResolver(t)
	rawConfig := map[string]interface{}{
		"version": latest.Version,
		"name":    "test",
		"imports": []interface{}{
			map[string]interface{}{"path": "enabled.yaml"},
			map[string]interface{}{"path": "missing-disabled.yaml", "enabled": false},
		},
	}

	resolved, err := ResolveImports(context.Background(), resolver, tempDir, rawConfig, log.Discard)
	assert.NilError(t, err)

	pipelines := resolved["pipelines"].(map[string]interface{})
	assert.Assert(t, pipelines["enabled"] != nil, "expected enabled import to be merged")
}

func newImportTestResolver(t *testing.T) variable.Resolver {
	t.Helper()

	resolver, err := variable.NewResolver(localcache.New(filepath.Join(t.TempDir(), "cache.yaml")), &variable.PredefinedVariableOptions{
		ConfigPath: filepath.Join(t.TempDir(), "devspace.yaml"),
	}, nil, log.Discard)
	assert.NilError(t, err)

	return resolver
}

func writeImportFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	assert.NilError(t, err)
}
