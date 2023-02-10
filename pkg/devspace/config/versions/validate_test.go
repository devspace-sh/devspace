package versions

import (
	"fmt"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gotest.tools/assert"
)

func TestValidateImageName(t *testing.T) {
	config := &latest.Config{
		Images: map[string]*latest.Image{
			"default": {
				Image: "localhost:5000/node",
			},
		},
	}
	err := validateImages(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Images: map[string]*latest.Image{
			"default": {
				Image: "localhost:5000/node:latest",
			},
		},
	}
	err = validateImages(config)
	assert.Error(t, err, "images.default.image 'localhost:5000/node:latest' can not have tag 'latest'")
}

func TestValidateHooks(t *testing.T) {
	config := &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events: []string{
					"after:deploy:my-deployment",
				},
				Container: &latest.HookContainer{
					ContainerName: "fakeContainer",
					LabelSelector: map[string]string{
						"app": "selectThisContainer",
					},
				},
				Command: "doSomething",
			},
		},
	}

	err := validateHooks(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events: []string{
					"after:deploy:my-deployment",
				},
				Container: &latest.HookContainer{
					ContainerName: "fakeContainer",
				},
				Command: "doSomething",
			},
		},
	}

	err = validateHooks(config)
	assert.Error(t, err, "hooks[0].container.containerName is defined but hooks[0].container.labelSelector is not defined")
}

func TestValidateDev(t *testing.T) {
	// test port forwarding
	localPort := int(8080)
	remotePort := int(9090)
	config := &latest.Config{
		Dev: map[string]*latest.DevPod{
			"somename": {
				Name:          "somename",
				ImageSelector: "selecMe",
				DevContainer: latest.DevContainer{
					Container: "fakeContainer",
				},
				Ports: []*latest.PortMapping{
					{
						Port: fmt.Sprintf("%v:%v", localPort, remotePort),
					},
				},
			},
		},
	}

	err := validateDev(config)
	assert.NilError(t, err)

	// test sync
	config = &latest.Config{
		Dev: map[string]*latest.DevPod{
			"somename": {
				Name: "somename",
				LabelSelector: map[string]string{
					"app": "MeApp",
				},
				DevContainer: latest.DevContainer{
					Container: "fakeContainers",
				},
			},
		},
	}

	err = validateDev(config)
	assert.NilError(t, err)

	// test replace pods
	config = &latest.Config{
		Dev: map[string]*latest.DevPod{
			"test": {
				LabelSelector: map[string]string{
					"app": "MeApp",
				},
				DevContainer: latest.DevContainer{
					Container: "fakeContainer",
				},
			},
		},
	}

	err = validateDev(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Dev: map[string]*latest.DevPod{
			"test": {
				DevContainer: latest.DevContainer{
					Container: "fakeContainer",
				},
			},
		},
	}

	err = validateDev(config)
	assert.Error(t, err, "dev.test: image selector and label selector are nil")

	// test devpod overwritten by devcontainer
	config = &latest.Config{
		Dev: map[string]*latest.DevPod{
			"somename": {
				Name: "somename",
				LabelSelector: map[string]string{
					"app": "MeApp",
				},
				DevContainer: latest.DevContainer{
					ReversePorts: []*latest.PortMapping{
						{
							Port: fmt.Sprintf("%v:%v", 8080, 8080),
						},
					},
				},
				Containers: map[string]*latest.DevContainer{
					"test": {
						Container: "test",
						ReversePorts: []*latest.PortMapping{
							{
								Port: fmt.Sprintf("%v:%v", 8081, 8081),
							},
						},
					},
				},
			},
		},
	}

	err = validateDev(config)
	assert.Error(t, err, "dev.somename.reversePorts will be overwritten by dev.somename.containers[test], please specify dev.somename.containers[test].reversePorts instead")
}
