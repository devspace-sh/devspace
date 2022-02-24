package kubectl

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
)

func Delete(ctx *devspacecontext.Context, deploymentName string) error {
	deploymentCache, ok := ctx.Config.RemoteCache().GetDeploymentCache(deploymentName)
	if !ok || deploymentCache.IsKubectl == false || len(deploymentCache.KubectlObjects) == 0 {
		ctx.Config.RemoteCache().DeleteDeploymentCache(deploymentName)
		return nil
	}

	for _, resource := range deploymentCache.KubectlObjects {
		_, err := ctx.KubeClient.GenericRequest(ctx.Context, &kubectl.GenericRequestOptions{
			Kind:       resource.Kind,
			APIVersion: resource.APIVersion,
			Name:       resource.Name,
			Namespace:  resource.Namespace,
			Method:     "delete",
		})
		if err != nil {
			ctx.Log.Errorf("error deleting %s %s: %v", resource.Kind, resource.Name, err)
		}
	}

	// Delete from cache
	ctx.Config.RemoteCache().DeleteDeploymentCache(deploymentName)
	return nil
}
