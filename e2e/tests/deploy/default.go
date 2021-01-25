package deploy

import (
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

//Test 1 - default
//1. deploy (without profile & var)
//2. deploy --force-build & check if rebuild
//3. deploy --force-deploy & check NO build but deployed
//4. deploy --force-dependencies & check NO build & check NO deployment but dependencies are deployed
//5. deploy --force-deploy --deployments=default,test2 & check NO build & only deployments deployed

// RunDefault runs the test for the default deploy test
func RunDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'deploy'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	client, err := f.NewKubeClientFromContext("", f.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// The client is saved in the factory ONCE for each sub test
	f.Client = client

	ts := testSuite{
		test{
			name: "1. deploy (without profile & var)",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
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
					Namespace: f.Namespace,
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
					Namespace: f.Namespace,
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

				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, "sh.helm.release.v1.root-app.v2")
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
					Namespace: f.Namespace,
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

				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, deployedDependencies...)
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
					Namespace: f.Namespace,
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

				shouldBeDeployed := "sh.helm.release.v1.root-app.v4"
				shouldNotBeDeployed := "sh.helm.release.v1.php-app.v5"

				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, shouldBeDeployed)
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.Errorf("expected deployment '%v' was not found", shouldBeDeployed)
				}

				wasDeployed, err = utils.LookForDeployment(f.Client, f.Namespace, shouldNotBeDeployed)
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

	err = beforeTest(f, logger, "tests/deploy/testdata/default")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'default' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("default", t.name, err, logger)
		if err != nil {
			return errors.Errorf("sub test 'default' of 'deploy' test failed: %s %v", f.GetLogContents(), err)
		}
	}

	return nil
}

func checkPortForwarding(f *customFactory, deployConfig *cmd.DeployCmd) error {
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
	err = utils.PortForwardAndPing(config, generatedConfig, f.Client, f.GetLog())
	if err != nil {
		return err
	}

	return nil
}
