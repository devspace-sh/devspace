package tanka

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
)

func Purge(ctx devspacecontext.Context, deploymentName string) error {
	deploymentCache, ok := ctx.Config().RemoteCache().GetDeployment(deploymentName)

	// if we have no cache, exit
	if !ok {
		return nil
	}

	tanka := NewTankaEnvironment(deploymentCache.Tanka.AppliedTankaConfig)
	return tanka.Delete(ctx)
}
