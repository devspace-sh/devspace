package minikube

import (
	"testing"
	
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	
	"gotest.tools/assert"
)

func TestIsMinikube(t *testing.T){
	isMinikubeVar = nil
	config := &latest.Config{
		Cluster: &latest.Cluster{
			KubeContext: ptr.String("minikube"),
		},
	}
	assert.Equal(t, true, IsMinikube(config), "Minikube config declared as not minikube")

}
