package context

import (
	context2 "context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/env"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
)

func NewContext(ctx context2.Context, variables map[string]interface{}, log log.Logger) Context {
	var err error
	workingDir, _ := RealWorkDir()
	if workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			panic(errors.Wrap(err, "get current working directory"))
		}
	}

	return &context{
		context:    ctx,
		workingDir: workingDir,
		runID:      strings.ToLower(randutil.GenerateRandomString(12)),
		environ:    env.NewVariableEnvProvider(expand.ListEnviron(os.Environ()...), env.ConvertMap(variables)),
		log:        log,
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
	return "", nil
}

type Context interface {
	// Context is the golang context to use
	Context() context2.Context

	// WorkingDir is the current working dir. Functions
	// that receive this context should prefer this working directory
	// instead of using the global one.
	WorkingDir() string

	// RunID is the current DevSpace run id, which differs in each
	// run of DevSpace. This can be used to save certain informations
	// during the run.
	RunID() string

	// Environ is the environment for command execution
	Environ() expand.Environ

	// Log is the currently used logger
	Log() log.Logger

	// Config is the loaded DevSpace config
	Config() config.Config

	// Dependencies are the loaded dependencies
	Dependencies() []types.Dependency

	// KubeClient is the kubernetes client
	KubeClient() kubectl.Client

	// IsDone checks if the context expired
	IsDone() bool

	// ResolvePath resolves a relative path according to the
	// current working directory
	ResolvePath(relPath string) string

	WithNewTomb() (Context, *tomb.Tomb)
	WithKubeClient(client kubectl.Client) Context
	WithWorkingDir(workingDir string) Context
	WithConfig(conf config.Config) Context
	WithDependencies(dependencies []types.Dependency) Context
	WithContext(ctx context2.Context) Context
	WithEnviron(environ expand.Environ) Context
	WithLogger(logger log.Logger) Context
	AsDependency(dependency types.Dependency) Context
}

type context struct {
	// context is the context to use
	context context2.Context

	// workingDir is the current working dir. Functions
	// that receive this context should prefer this working directory
	// instead of using the global one.
	workingDir string

	// runID is the current DevSpace run id, which differs in each
	// run of DevSpace. This can be used to save certain informations
	// during the run.
	runID string

	// config is the loaded DevSpace config
	config config.Config

	// dependencies are the loaded dependencies
	dependencies []types.Dependency

	// kubeClient is the kubernetes client
	kubeClient kubectl.Client

	// environ is the environment provider used for executing a command
	environ expand.Environ

	// log is the currently used logger
	log log.Logger
}

func (c *context) Environ() expand.Environ {
	return c.environ
}

func (c *context) Context() context2.Context {
	return c.context
}

func (c *context) WorkingDir() string {
	return c.workingDir
}

func (c *context) RunID() string {
	return c.runID
}

func (c *context) Config() config.Config {
	return c.config
}

func (c *context) Dependencies() []types.Dependency {
	return c.dependencies
}

func (c *context) KubeClient() kubectl.Client {
	return c.kubeClient
}

func (c *context) Log() log.Logger {
	return c.log
}

func (c *context) IsDone() bool {
	select {
	case <-c.context.Done():
		return true
	default:
	}

	return false
}

func (c *context) WithEnviron(environ expand.Environ) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.environ = environ
	return &n
}

func (c *context) WithNewTomb() (Context, *tomb.Tomb) {
	if c == nil {
		return nil, nil
	}

	var t *tomb.Tomb
	n := *c
	t, n.context = tomb.WithContext(c.context)
	return &n, t
}

func (c *context) ResolvePath(relPath string) string {
	if relPath == "" {
		return c.workingDir
	}

	relPath = filepath.ToSlash(relPath)
	if filepath.IsAbs(relPath) {
		return path.Clean(relPath)
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		if relPath == "~" {
			return homeDir
		} else if strings.HasPrefix(relPath, "~/") {
			return path.Clean(filepath.Join(homeDir, relPath[2:]))
		}
	}

	outPath := path.Join(filepath.ToSlash(c.workingDir), relPath)
	if !filepath.IsAbs(outPath) {
		return c.workingDir
	}

	return outPath
}

func (c *context) WithKubeClient(client kubectl.Client) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.kubeClient = client
	return &n
}

func (c *context) WithWorkingDir(workingDir string) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.workingDir = workingDir
	return &n
}

func (c *context) WithConfig(conf config.Config) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.config = conf
	return &n
}

func (c *context) WithDependencies(dependencies []types.Dependency) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.dependencies = dependencies
	return &n
}

func (c *context) WithContext(ctx context2.Context) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.context = ctx
	return &n
}

func (c *context) WithLogger(logger log.Logger) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.log = logger
	return &n
}

func (c *context) AsDependency(dependency types.Dependency) Context {
	if c == nil {
		return nil
	}

	n := *c
	n.workingDir = dependency.Path()
	n.kubeClient = dependency.KubeClient()
	n.config = dependency.Config()
	n.dependencies = dependency.Children()
	n.environ = env.NewVariableEnvProvider(c.environ, env.ConvertMap(n.config.Variables()))
	return &n
}
