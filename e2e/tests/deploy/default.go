package deploy

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"path/filepath"
)

//Test 1 - default
//1. deploy (without profile & var)
//2. deploy --force-build & check if rebuild
//3. deploy --force-deploy & check NO build but deployed
//4. deploy --force-dependencies & check NO build & check NO deployment but dependencies are deployed
//5. deploy --force-deploy --deployments=default,test2 & check NO build & only deployments deployed

// RunDefault runs the test for the default deploy test
func RunDefault(f *customFactory) error {
	log.GetInstance().Info("Run Default")

	ts := testSuite{
		test{
			name: "1. deploy (without profile & var)",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
			},
			postCheck: func(f *customFactory, t *test) error {
				err := checkPortForwarding(f, t.deployConfig)
				if err != nil {
					return err
				}

				return nil
			},
		},
		test{
			name: "2. deploy --force-build & check if rebuild",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
				ForceBuild: true,
			},
			postCheck: func(f *customFactory, t *test) error {
				imagesExpected := 1
				imagesCount := len(f.builtImages)
				if imagesCount != imagesExpected {
					return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
				}

				return nil
			},
		},
		test{
			name: "3. deploy --force-deploy & check NO build but deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
				ForceDeploy: true, // Only forces to redeploy deployments
			},
			postCheck: func(f *customFactory, t *test) error {
				imagesExpected := 0
				imagesCount := len(f.builtImages)
				if imagesCount != imagesExpected {
					return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
				}

				client, err := f.NewKubeClientFromContext(t.deployConfig.KubeContext, t.deployConfig.Namespace, t.deployConfig.SwitchContext)
				if err != nil {
					return errors.Errorf("Unable to create new kubectl client: %v", err)
				}

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, "sh.helm.release.v1.root-app.v2")
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected deployment 'sh.helm.release.v1.root-app.v2' was not found")
				}

				return nil
			},
		},
		test{
			name: "4. deploy --force-dependencies & check NO build & check NO deployment but dependencies are deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
				ForceDeploy:       true,
				ForceDependencies: true,
			},
			postCheck: func(f *customFactory, t *test) error {
				// No build
				imagesExpected := 0
				imagesCount := len(f.builtImages)
				if imagesCount != imagesExpected {
					return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
				}

				deployedDependencies := []string{"sh.helm.release.v1.dependency1.v2", "sh.helm.release.v1.dependency2.v2"}
				client, err := f.NewKubeClientFromContext(t.deployConfig.KubeContext, t.deployConfig.Namespace, t.deployConfig.SwitchContext)
				if err != nil {
					return errors.Errorf("Unable to create new kubectl client: %v", err)
				}

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, deployedDependencies...)
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected dependency deployment was not found")
				}

				return nil
			},
		},
		test{
			name: "5. deploy --force-deploy --deployments=default,test2 & check NO build & only deployments deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
				ForceDeploy: true,
				Deployments: "root-app",
			},
			postCheck: func(f *customFactory, t *test) error {
				// No build
				imagesExpected := 0
				imagesCount := len(f.builtImages)
				if imagesCount != imagesExpected {
					return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
				}

				client, err := f.NewKubeClientFromContext(t.deployConfig.KubeContext, t.deployConfig.Namespace, t.deployConfig.SwitchContext)
				if err != nil {
					return errors.Errorf("Unable to create new kubectl client: %v", err)
				}

				shouldBeDeployed := "sh.helm.release.v1.root-app.v4"
				shouldNotBeDeployed := "sh.helm.release.v1.php-app.v5"

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, shouldBeDeployed)
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.Errorf("expected deployment '%v' was not found", shouldBeDeployed)
				}

				wasDeployed, err = utils.LookForDeployment(client, f.namespace, shouldNotBeDeployed)
				if err != nil {
					return err
				}
				if wasDeployed {
					return errors.Errorf("deployment '%v' should not be found", shouldNotBeDeployed)
				}

				return nil
			},
		},
	}

	client, err := f.NewKubeClientFromContext("", f.namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// At last, we delete the current namespace
	defer utils.DeleteNamespaceAndWait(client, f.namespace)

	testDir := filepath.FromSlash("tests/deploy/testdata/default")

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	defer utils.DeleteTempAndResetWorkingDir(dirPath, f.pwd)

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath)
	if err != nil {
		return err
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("default", t.name, err)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkPortForwarding(f *customFactory, deployConfig *cmd.DeployCmd) error {
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Load generated config
	generatedConfig, err := f.NewConfigLoader(nil, nil).Generated()
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}

	// Add current kube context to context
	configOptions := deployConfig.ToConfigOptions()
	config, err := f.NewConfigLoader(configOptions, f.GetLog()).Load()
	if err != nil {
		return err
	}

	// Port-forwarding
	err = utils.PortForwardAndPing(config, generatedConfig, client)
	if err != nil {
		return err
	}

	return nil
}
