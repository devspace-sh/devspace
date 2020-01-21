package deploy

import (
	"path/filepath"
	"time"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testSuite []test

type test struct {
	name         string
	deployConfig *cmd.DeployCmd
	postCheck    func(f *customFactory, t *test) error
}

type customFactory struct {
	*utils.BaseCustomFactory
	ctrl        build.Controller
	builtImages map[string]string
}

// NewBuildController implements interface
func (c *customFactory) NewBuildController(config *latest.Config, cache *generated.CacheConfig, client kubectl.Client) build.Controller {
	c.ctrl = build.NewController(config, cache, client)
	return c
}
func (c *customFactory) Build(options *build.Options, log log.Logger) (map[string]string, error) {
	m, err := c.ctrl.Build(options, log)
	c.builtImages = m

	return m, err
}

type Runner struct{}

var RunNew = &Runner{}

func (r *Runner) SubTests() []string {
	subTests := []string{}
	for k := range availableSubTests {
		subTests = append(subTests, k)
	}

	return subTests
}

var availableSubTests = map[string]func(factory *customFactory, logger log.Logger) error{
	"default": RunDefault,
	"profile": RunProfile,
	"kubectl": RunKubectl,
	"helm":    RunHelm,
	"helm-v2": RunHelmV2,
}

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run 'deploy' test")

	// Populates the tests to run with all the available sub tests if no sub tests are specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	f := &customFactory{
		BaseCustomFactory: &utils.BaseCustomFactory{
			Namespace: ns,
			Pwd:       pwd,
			Verbose:   verbose,
			Timeout:   timeout,
		},
	}

	// Runs the tests
	for _, subTestName := range subTests {
		f.ResetLog()
		c1 := make(chan error)

		go func() {
			err := func() error {
				f.Namespace = utils.GenerateNamespaceName("test-deploy-" + subTestName)
				err := availableSubTests[subTestName](f, logger)
				utils.PrintTestResult("deploy", subTestName, err, logger)
				if err != nil {
					return err
				}

				return nil
			}()
			c1 <- err
		}()

		select {
		case err := <-c1:
			if err != nil {
				return err
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			return errors.Errorf("Timeout error - the test did not return within the specified timeout of %v seconds: %s", timeout, f.GetLogContents())
		}
	}

	return nil
}

// Used by the different sub tests
func runTest(f *customFactory, t *test) error {
	// 1. Create kube client
	// 2. Deploy config
	// 3. Analyze pods
	// 4. Optional - Run the postCheck

	// 1. Create kube client
	client, err := f.NewKubeClientFromContext(t.deployConfig.KubeContext, t.deployConfig.Namespace, t.deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	f.Client = client

	// 2. Deploy config
	err = t.deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// 3. Analyze pods
	err = utils.AnalyzePods(client, f.Namespace, f.GetLog())
	if err != nil {
		return err
	}

	// 4. Optional - Run the postCheck
	if t.postCheck != nil {
		err = t.postCheck(f, t)
		if err != nil {
			return err
		}
	}

	return nil
}

func testPurge(f *customFactory) error {
	purgeCmd := &cmd.PurgeCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
		},
	}

	err := purgeCmd.Run(f, nil, nil)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext("", f.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	for start := time.Now(); time.Since(start) < time.Second*30; {
		p, _ := client.KubeClient().CoreV1().Pods(f.Namespace).List(metav1.ListOptions{})

		if len(p.Items) == 0 || len(p.Items) == 1 && p.Items[0].Status.ContainerStatuses[0].Name == "tiller" {
			return nil
		}
	}

	p, _ := client.KubeClient().CoreV1().Pods(f.Namespace).List(metav1.ListOptions{})
	return errors.Errorf("purge command failed, expected 1 (tiller) pod but found %v", len(p.Items))
}

func beforeTest(f *customFactory, logger log.Logger, testDir string) error {
	testDir = filepath.FromSlash(testDir)

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.DirPath = dirPath

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath, f.GetLog())
	if err != nil {
		return err
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.DirPath, f.Pwd, f.GetLog())
	utils.DeleteNamespace(f.Client, f.Namespace)
}
