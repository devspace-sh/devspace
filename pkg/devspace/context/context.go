package context

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func NewContext(ctx context.Context, log log.Logger) *Context {
	var err error
	workingDir, _ := RealWorkDir()
	if workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			panic(errors.Wrap(err, "get current working directory"))
		}
	}

	return &Context{
		Context:    ctx,
		WorkingDir: workingDir,
		RunID:      strings.ToLower(randutil.GenerateRandomString(12)),
		Log:        log,
	}
}

func RealWorkDir() (string, error) {
	if runtime.GOOS == "darwin" {
		if pwd, present := os.LookupEnv("PWD"); present {
			os.Unsetenv("PWD")
			defer os.Setenv("PWD", pwd)
		}
		return os.Getwd()
	}
	return ".", nil
}

type Context struct {
	// Context is the context to use
	Context context.Context

	// WorkingDir is the current working dir. Functions
	// that receive this context should prefer this working directory
	// instead of using the global one.
	WorkingDir string

	// RunID is the current DevSpace run id, which differs in each
	// run of DevSpace. This can be used to save certain informations
	// during the run.
	RunID string

	// Config is the loaded DevSpace config
	Config config.Config

	// Dependencies are the loaded dependencies
	Dependencies []types.Dependency

	// KubeClient is the kubernetes client
	KubeClient kubectl.Client

	// Log is the currently used logger
	Log log.Logger
}

func (c *Context) IsDone() bool {
	select {
	case <-c.Context.Done():
		return true
	default:
	}

	return false
}

func (c *Context) WithNewTomb() (*Context, *tomb.Tomb) {
	if c == nil {
		return nil, nil
	}

	var t *tomb.Tomb
	n := *c
	t, n.Context = tomb.WithContext(c.Context)
	return &n, t
}

func (c *Context) ToOriginalRelativePath(absPath string) string {
	relPath, err := filepath.Rel(c.WorkingDir, absPath)
	if err != nil {
		c.Log.Debugf("Error computing original relative path: %v", err)
		return absPath
	}
	return relPath
}

func (c *Context) ResolvePath(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	if filepath.IsAbs(relPath) {
		return path.Clean(relPath)
	}

	return path.Join(filepath.ToSlash(c.WorkingDir), relPath)
}

func (c *Context) WithKubeClient(client kubectl.Client) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.KubeClient = client
	return &n
}

func (c *Context) WithWorkingDir(workingDir string) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.WorkingDir = workingDir
	return &n
}

func (c *Context) WithConfig(conf config.Config) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.Config = conf
	return &n
}

func (c *Context) WithDependencies(dependencies []types.Dependency) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.Dependencies = dependencies
	return &n
}

func (c *Context) WithContext(ctx context.Context) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.Context = ctx
	return &n
}

func (c *Context) WithLogger(logger log.Logger) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.Log = logger
	return &n
}

func (c *Context) AsDependency(dependency types.Dependency) *Context {
	if c == nil {
		return nil
	}

	n := *c
	n.WorkingDir = dependency.Path()
	n.KubeClient = dependency.KubeClient()
	n.Config = dependency.Config()
	n.Dependencies = dependency.Children()
	return &n
}
