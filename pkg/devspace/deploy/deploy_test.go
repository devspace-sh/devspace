package deploy

import (
	"context"
	"testing"

	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakekube "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type renderTestCase struct {
	name string

	deploymentConfigs map[string]*latest.DeploymentConfig
	options           *Options
	deploymentNames   []string
	expectedErr       string
}

func TestRender(t *testing.T) {
	testCases := []renderTestCase{
		{
			name: "Skip deployment",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"skippedDeployment": {
					Name: "skippedDeployment",
				},
			},
			options: &Options{
				SkipDeploy: true,
				Render:     true,
			},
			deploymentNames: []string{"unskippedDeployment"},
		},
		{
			name: "No deployment method",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"noMethod": {
					Name: "noMethod",
				},
			},
			deploymentNames: []string{"noMethod"},
			options: &Options{
				Render: true,
			},
			expectedErr: "error deploying: deployment noMethod has no deployment method",
		},
		{
			name: "Render with kubectl",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"kubectlRender": {
					Name: "kubectlRender",
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{},
					},
				},
			},
			options: &Options{
				Render: true,
			},
			deploymentNames: []string{"kubectlRender"},
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}
		config := &latest.Config{
			Deployments: testCase.deploymentConfigs,
		}
		controller := NewController()

		if testCase.options == nil {
			testCase.options = &Options{}
		}

		conf := config2.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			config,
			&localcache.LocalCache{},
			remotecache.NewCache("", "test"),
			map[string]interface{}{},
			constants.DefaultConfigPath)
		devCtx := devspacecontext.NewContext(context.TODO(), nil, log.Discard).WithKubeClient(kubeClient).WithConfig(conf)
		err := controller.Deploy(devCtx, testCase.deploymentNames, testCase.options)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type deployTestCase struct {
	name string

	deploymentConfigs map[string]*latest.DeploymentConfig
	options           *Options
	deploymentNames   []string
	expectedErr       string
}

func TestDeploy(t *testing.T) {
	testCases := []deployTestCase{
		{
			name: "Skip deployment",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"skippedDeployment": {
					Name: "skippedDeployment",
				},
			},
			options: &Options{
				SkipDeploy: true,
				Render:     true,
			},
			deploymentNames: []string{"unskippedDeployment"},
		},
		{
			name: "No deployment method",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"noMethod": {
					Name: "noMethod",
				},
			},
			deploymentNames: []string{"noMethod"},
			options: &Options{
				Render: true,
			},
			expectedErr: "error deploying: deployment noMethod has no deployment method",
		},
		{
			name: "Deploy with kubectl",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"kubectlDeploy": {
					Name: "kubectlDeploy",
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{},
					},
				},
			},
			options: &Options{
				Render: true,
			},
			deploymentNames: []string{"kubectlDeploy"},
		},
		{
			name: "Deploy concurrently",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"concurrentDeploy1": {
					Name: "concurrentDeploy1",
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{},
					},
				},
				"concurrentDeploy2": {
					Name: "concurrentDeploy2",
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{},
					},
				},
			},
			options: &Options{
				Render:     true,
				Sequential: false,
			},
			deploymentNames: []string{"concurrentDeploy1", "concurrentDeploy2"},
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}
		config := &latest.Config{
			Deployments: testCase.deploymentConfigs,
		}
		controller := NewController()

		if testCase.options == nil {
			testCase.options = &Options{}
		}

		conf := config2.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			config,
			&localcache.LocalCache{},
			remotecache.NewCache("", "test"),
			map[string]interface{}{},
			constants.DefaultConfigPath)
		devCtx := devspacecontext.NewContext(context.TODO(), nil, log.Discard).WithKubeClient(kubeClient).WithConfig(conf)
		err := controller.Deploy(devCtx, testCase.deploymentNames, testCase.options)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type purgeTestCase struct {
	name string

	deploymentConfigs map[string]*latest.DeploymentConfig
	deploymentNames   []string

	expectedErr string
}

func TestPurge(t *testing.T) {
	testCases := []purgeTestCase{
		{
			name: "Skip deployment",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"skippedDeployment": {
					Name: "skippedDeployment",
				},
			},
			deploymentNames: []string{"unskippedDeployment"},
		},
		{
			name: "No deployment method",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"noMethod": {
					Name: "noMethod",
				},
			},
			deploymentNames: []string{"noMethod"},
			expectedErr:     "error purging: deployment noMethod has no deployment method",
		},
		{
			name: "Deploy with kubectl",
			deploymentConfigs: map[string]*latest.DeploymentConfig{
				"kubectlDeploy": {
					Name: "kubectlDeploy",
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{},
					},
				},
			}, deploymentNames: []string{"kubectlDeploy"},
		},
	}

	for _, testCase := range testCases {
		kube := fake.NewSimpleClientset()
		kubeClient := &fakekube.Client{
			Client: kube,
		}
		config := &latest.Config{
			Deployments: testCase.deploymentConfigs,
		}

		controller := NewController()
		conf := config2.NewConfig(map[string]interface{}{},
			map[string]interface{}{},
			config,
			&localcache.LocalCache{},
			remotecache.NewCache("", "test"),
			map[string]interface{}{},
			constants.DefaultConfigPath)
		devCtx := devspacecontext.NewContext(context.TODO(), nil, log.Discard).WithKubeClient(kubeClient).WithConfig(conf)
		err := controller.Purge(devCtx, testCase.deploymentNames, &PurgeOptions{})

		assert.NilError(t, err)
	}
}
