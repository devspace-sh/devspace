package add

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
 
	"gotest.tools/assert"
)

func TestAddExistingDeployment(t *testing.T) {
	t.Skip("Untestable because log.Fatal is uncatchable")
	deployment := &deploymentCmd{

	}

	config := &latest.Config{}
	config.Deployments = &[]*latest.DeploymentConfig{}
	(*config.Deployments)[0] = &latest.DeploymentConfig{
		Name: ptr.String("exists"),
	}

	//Method should panic, preparing recovery
	paniced := false
	defer func(){
		paniced = true
		recover()
	}()

	deployment.RunAddDeployment(nil, []string{"exists"})

	assert.Equal(t, true, paniced, "Method didn't panic")
}
