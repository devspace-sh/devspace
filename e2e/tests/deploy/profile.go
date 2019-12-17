package deploy

import (
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

//Test 2 - profile
//1. deploy --profile=bla --var var1=two --var var2=three
//2. deploy --profile=bla --var var1=two --var var2=three --force-build & check if rebuild
//3. deploy --profile=bla --var var1=two --var var2=three --force-deploy & check NO build but deployed
//4. deploy --profile=bla --var var1=two --var var2=three --force-dependencies & check NO build & check NO deployment but dependencies are deployed
//5. deploy --profile=bla --var var1=two --var var2=three --force-deploy --deployments=default,test2 & check NO build & only deployments deployed

// RunProfile runs the test for the default profile test
func RunProfile(f *customFactory) error {
	log.GetInstance().Info("Run Profile")

	ts := testSuite{
		test{
			name: "1. deploy --profile=bla --var var1=two --var var2=three",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
				},
			},
			postCheck: func(f *customFactory, t *test) error {
				client, err := f.NewKubeClientFromContext(t.deployConfig.KubeContext, t.deployConfig.Namespace, t.deployConfig.SwitchContext)
				if err != nil {
					return errors.Errorf("Unable to create new kubectl client: %v", err)
				}

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, "sh.helm.release.v1.service-2.v1")
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected deployment 'sh.helm.release.v1.service-2.v1' was not found")
				}

				return nil
			},
		},
		test{
			name: "2. deploy --profile=bla --var var1=two --var var2=three --force-build & check if rebuild",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
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
			name: "3. deploy --profile=bla --var var1=two --var var2=three --force-deploy & check NO build but deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
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

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, "sh.helm.release.v1.service-2.v3")
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected deployment 'sh.helm.release.v1.service-2.v3' was not found")
				}

				return nil
			},
		},
		test{
			name: "4. deploy --profile=bla --var var1=two --var var2=three --force-dependencies & check NO build & check NO deployment but dependencies are deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
				},
				ForceDeploy:       true,
				ForceDependencies: true,
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

				wasDeployed, err := utils.LookForDeployment(client, f.namespace, "sh.helm.release.v1.dependency1.v3")
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected deployment 'sh.helm.release.v1.dependency1.v3' was not found")
				}

				return nil
			},
		},
		test{
			name: "5. deploy --profile=bla --var var1=two --var var2=three --force-deploy --deployments=default,test2 & check NO build & only deployments deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
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
				shouldNotBeDeployed := "sh.helm.release.v1.service-2.v5"

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

	testDir := filepath.FromSlash("tests/deploy/testdata/profile")

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
		utils.PrintTestResult("profile", t.name, err)
		if err != nil {
			return err
		}
	}

	return nil
}
