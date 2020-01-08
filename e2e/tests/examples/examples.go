package examples

import (
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

type customFactory struct {
	*factory.DefaultFactoryImpl
	namespace string
	pwd       string
	verbose   bool
	timeout   int

	cacheLogger log.Logger
	dirPath     string
	client      kubectl.Client
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.cacheLogger
}

var availableSubTests = map[string]func(factory *customFactory, logger log.Logger) error{
	"quickstart":         RunQuickstart,
	"kustomize":          RunKustomize,
	"profiles":           RunProfiles,
	"microservices":      RunMicroservices,
	"minikube":           RunMinikube,
	"quickstart-kubectl": RunQuickstartKubectl,
	"php-mysql":          RunPhpMysql,
	"dependencies":       RunDependencies,
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

func (r *Runner) Run(subTests []string, ns string, pwd string, logger log.Logger, verbose bool, timeout int) error {
	logger.Info("Run 'examples' test")

	// Populates the tests to run with all the available sub tests if no sub tests in specified
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
		myFactory.namespace = utils.GenerateNamespaceName("test-examples-" + subTestName)
		err := availableSubTests[subTestName](myFactory, logger)
		utils.PrintTestResult("examples", subTestName, err, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func RunTest(f *customFactory, deployConfig *cmd.DeployCmd) error {
	if deployConfig == nil {
		deployConfig = &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.namespace,
				NoWarn:    true,
			},
			ForceBuild:  true,
			ForceDeploy: true,
			SkipPush:    true,
		}
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	f.client = client

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.namespace, f.cacheLogger)
	if err != nil {
		return err
	}

	// Load generated config
	generatedConfig, err := f.NewConfigLoader(nil, nil).Generated()
	if err != nil {
		return errors.Errorf("Error loading generated.yaml: %v", err)
	}

	// Add current kube context to context
	configOptions := deployConfig.ToConfigOptions()
	config, err := f.NewConfigLoader(configOptions, f.cacheLogger).Load()
	if err != nil {
		return err
	}

	// Port-forwarding
	err = utils.PortForwardAndPing(config, generatedConfig, client, f.cacheLogger)
	if err != nil {
		return err
	}

	return nil
}

func beforeTest(f *customFactory, testDir string) error {
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
