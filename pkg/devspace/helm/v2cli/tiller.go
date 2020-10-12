package v2cli

import (
	"context"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/abstractcli"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(config *latest.Config, client kubectl.Client, tillerNamespace string) bool {
	deployment, err := client.KubeClient().AppsV1().Deployments(tillerNamespace).Get(context.TODO(), abstractcli.TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	return true
}
