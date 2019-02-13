package configutil

import (
	"testing"

	v1 "github.com/covexo/devspace/pkg/devspace/config/versions/latest"
	"github.com/covexo/devspace/pkg/util/ptr"
)

func TestSimpleMerge(t *testing.T) {
	apiServer := ptr.String("testApiServer")
	deployment1 := ptr.String("testDeployment1")
	deployment2 := ptr.String("testDeployment2")
	version := ptr.String("testVersion")

	object1 := &v1.Config{
		Version: ptr.String("oldVersion"),
		DevSpace: &v1.DevSpaceConfig{
			Deployments: &[]*v1.DeploymentConfig{
				&v1.DeploymentConfig{
					Name: ptr.String("oldDeployment"),
				},
			},
		},
	}

	object2 := &v1.Config{
		Version: version,
		DevSpace: &v1.DevSpaceConfig{
			Deployments: &[]*v1.DeploymentConfig{
				&v1.DeploymentConfig{
					Name: deployment1,
				},
				&v1.DeploymentConfig{
					Name: deployment2,
				},
			},
		},
		Cluster: &v1.Cluster{
			APIServer: apiServer,
		},
	}

	// Merge object2 in object1
	Merge(&object1, object2)

	if object1.Version == nil || object1.Version != version {
		t.Fatal("Version is not equal")
	}
	if object1.Cluster == nil || object1.Cluster.APIServer == nil || object1.Cluster.APIServer != apiServer {
		t.Fatal("APIServer is not equal")
	}
	if object1.DevSpace == nil || object1.DevSpace.Deployments == nil || len(*object1.DevSpace.Deployments) != 2 || (*object1.DevSpace.Deployments)[0].Name != deployment1 || (*object1.DevSpace.Deployments)[1].Name != deployment2 {
		t.Fatal("Deployments are not correct")
	}
}
