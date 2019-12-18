package examples

import (
	"fmt"
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakelog "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/pkg/errors"
)

type customFactory struct {
	*factory.DefaultFactoryImpl
	namespace string
	pwd       string

	FakeLogger *fakelog.FakeLogger
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.FakeLogger
}

var availableSubTests = map[string]func(factory *customFactory) error{
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

func (r *Runner) Run(subTests []string, ns string, pwd string) error {
	// Populates the tests to run with all the available sub tests if no sub tests in specified
	if len(subTests) == 0 {
		for subTestName := range availableSubTests {
			subTests = append(subTests, subTestName)
		}
	}

	myFactory := &customFactory{
		namespace: ns,
		pwd:       pwd,
	}
	myFactory.FakeLogger = fakelog.NewFakeLogger()

	// Runs the tests
	for _, subTestName := range subTests {
		err := availableSubTests[subTestName](myFactory)
		utils.PrintTestResult("examples", subTestName, err)
		if err != nil {
			return err
		}
	}

	return nil
}

func RunTest(f *customFactory, dir string, deployConfig *cmd.DeployCmd) error {
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

	err := utils.ChangeWorkingDir(f.pwd + "/../examples/" + dir)
	if err != nil {
		return err
	}
	fmt.Println("A")
	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}
	fmt.Println("B")

	// At last, we delete the current namespace
	defer utils.DeleteNamespaceAndWait(client, deployConfig.Namespace)

	err = deployConfig.Run(f, nil, nil)
	fmt.Printf("After deployConfig.Run: %v\n", err)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.namespace)
	if err != nil {
		return err
	}

	fmt.Println("D")


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
