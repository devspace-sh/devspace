package context

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
)

func NewContext() *Context {
	return &Context{}
}

type Context struct {
	// Context is the context to use
	Context context.Context

	// Config is the loaded DevSpace config
	Config config.Config

	// Dependencies are the loaded dependencies
	Dependencies []types.Dependency

	// KubeClient is the kubernetes client
	KubeClient kubectl.Client

	// Log is the currently used logger
	Log log.Logger
}

func (c *Context) WithContext(ctx context.Context) *Context {
	if c == nil {
		return nil
	}

	return &Context{
		Context:      ctx,
		Config:       c.Config,
		Dependencies: c.Dependencies,
		KubeClient:   c.KubeClient,
		Log:          c.Log,
	}
}

func (c *Context) WithLogger(logger log.Logger) *Context {
	if c == nil {
		return nil
	}

	return &Context{
		Context:      c.Context,
		Config:       c.Config,
		Dependencies: c.Dependencies,
		KubeClient:   c.KubeClient,
		Log:          logger,
	}
}
