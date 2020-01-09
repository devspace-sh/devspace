package deploy

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//Test 2 - profile
//1. deploy --profile=bla --var var1=two --var var2=three
//2. deploy --profile=bla --var var1=two --var var2=three --force-build & check if rebuild
//3. deploy --profile=bla --var var1=two --var var2=three --force-deploy & check NO build but deployed
//4. deploy --profile=bla --var var1=two --var var2=three --force-dependencies & check NO build & check NO deployment but dependencies are deployed
//5. deploy --profile=bla --var var1=two --var var2=three --force-deploy --deployments=default,test2 & check NO build & only deployments deployed

// RunProfile runs the test for the default profile test
func RunProfile(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	var buffString string
	buffString = buff.String()

	if f.Verbose {
		f.cacheLogger = logger
		buffString = ""
	}

	logger.Info("Run sub test 'profile' of test 'deploy'")
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
			name: "1. deploy --profile=bla --var var1=two --var var2=three",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
					NoWarn:    true,
					Profile:   "dev-service2-only",
					Vars:      []string{"NAME=service-2"},
				},
			},
			postCheck: func(f *customFactory, t *test) error {
				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, "sh.helm.release.v1.service-2.v1")
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
					Namespace: f.Namespace,
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
					Namespace: f.Namespace,
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

				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, "sh.helm.release.v1.service-2.v3")
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
					Namespace: f.Namespace,
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

				wasDeployed, err := utils.LookForDeployment(f.Client, f.Namespace, "sh.helm.release.v1.dependency1.v2")
				if err != nil {
					return err
				}
				if !wasDeployed {
					return errors.New("expected deployment 'sh.helm.release.v1.dependency1.v2' was not found")
				}

				return nil
			},
		},
		test{
			name: "5. deploy --profile=bla --var var1=two --var var2=three --force-deploy --deployments=default,test2 & check NO build & only deployments deployed",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
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

				shouldBeDeployed := "sh.helm.release.v1.root-app.v4"
				shouldNotBeDeployed := "sh.helm.release.v1.service-2.v5"

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

	err = beforeTest(f, logger, "tests/deploy/testdata/profile")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'profile' of 'deploy' test failed: %s %v", buffString, err)
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("profile", t.name, err, logger)
		if err != nil {
			return errors.Errorf("sub test 'profile' of 'deploy' test failed: %s %v", buffString, err)
		}
	}

	return nil
}
