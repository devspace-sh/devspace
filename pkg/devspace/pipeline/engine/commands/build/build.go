package build

import (
	flags "github.com/jessevdk/go-flags"
	"github.com/loft-sh/devspace/pkg/devspace/build"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type BuildOptions struct {
	SkipPush                  bool `long:"skip-push" description:"Skip pushing"`
	SkipPushOnLocalKubernetes bool `long:"skip-push-on-local-kubernetes" description:"Skip pushing"`
	ForceRebuild              bool `long:"force-rebuild" description:"Skip pushing"`
	Sequential                bool `long:"sequential" description:"Skip pushing"`

	MaxConcurrentBuilds int `long:"maxConcurrentBuilds" description:"A pointer to an integer"`
}

func Build(configInterface config.Config, dependencies []types.Dependency, client kubectl.Client, logger log.Logger, args []string) error {
	options := &BuildOptions{}
	args, err := flags.ParseArgs(&options, args)
	if err != nil {
		return errors.Wrap(err, "parse args")
	}

	_, err = build.NewController(configInterface, dependencies, client).Build(&build.Options{
		SkipPush:                  options.SkipPush,
		SkipPushOnLocalKubernetes: options.SkipPushOnLocalKubernetes,
		ForceRebuild:              options.ForceRebuild,
		Sequential:                options.Sequential,
		MaxConcurrentBuilds:       options.MaxConcurrentBuilds,
	}, logger)
	return err
}
