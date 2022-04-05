package kubectl

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
)

func Delete(ctx devspacecontext.Context, deploymentName string) error {
	deploymentCache, ok := ctx.Config().RemoteCache().GetDeployment(deploymentName)
	if !ok || deploymentCache.Kubectl == nil || len(deploymentCache.Kubectl.Objects) == 0 {
		return nil
	}

	for _, resource := range deploymentCache.Kubectl.Objects {
		_, err := ctx.KubeClient().GenericRequest(ctx.Context(), &kubectl.GenericRequestOptions{
			Kind:       resource.Kind,
			APIVersion: resource.APIVersion,
			Name:       resource.Name,
			Namespace:  resource.Namespace,
			Method:     "delete",
		})
		if err != nil {
			ctx.Log().Errorf("error deleting %s %s: %v", resource.Kind, resource.Name, err)
		}
	}
	return nil
}
