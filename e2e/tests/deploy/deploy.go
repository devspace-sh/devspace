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
	"github.com/devspace-cloud/devspace/pkg/util/factory"
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
	*factory.DefaultFactoryImpl
	ctrl build.Controller

	verbose     bool
	timeout     int
	namespace   string
	pwd         string
	builtImages map[string]string

	client      kubectl.Client
	cacheLogger log.Logger
	dirPath     string
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.cacheLogger
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

	myFactory := &customFactory{
		namespace: ns,
		pwd:       pwd,
		verbose:   verbose,
		timeout:   timeout,
	}

	// Runs the tests
	for _, subTestName := range subTests {
		myFactory.namespace = utils.GenerateNamespaceName("test-deploy-" + subTestName)
		err := availableSubTests[subTestName](myFactory, logger)
		utils.PrintTestResult("deploy", subTestName, err, logger)
		if err != nil {
			return err
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

	f.client = client

	// 2. Deploy config
	err = t.deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// 3. Analyze pods
	err = utils.AnalyzePods(client, f.namespace, f.cacheLogger)
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
			Namespace: f.namespace,
			NoWarn:    true,
		},
	}

	err := purgeCmd.Run(f, nil, nil)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext("", f.namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	for start := time.Now(); time.Since(start) < time.Second*30; {
		p, _ := client.KubeClient().CoreV1().Pods(f.namespace).List(metav1.ListOptions{})

		if len(p.Items) == 0 || len(p.Items) == 1 && p.Items[0].Status.ContainerStatuses[0].Name == "tiller" {
			return nil
		}
	}

	p, _ := client.KubeClient().CoreV1().Pods(f.namespace).List(metav1.ListOptions{})
	return errors.Errorf("purge command failed, expected 1 (tiller) pod but found %v", len(p.Items))
}

func beforeTest(f *customFactory, logger log.Logger, testDir string) error {
	testDir = filepath.FromSlash(testDir)

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	f.dirPath = dirPath

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath, f.cacheLogger)
	if err != nil {
		return err
	}

	return nil
}

func afterTest(f *customFactory) {
	utils.DeleteTempAndResetWorkingDir(f.dirPath, f.pwd, f.cacheLogger)
	utils.DeleteNamespace(f.client, f.namespace)
}
