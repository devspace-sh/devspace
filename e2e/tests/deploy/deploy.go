package deploy

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/build"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakelog "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
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

	namespace   string
	pwd         string
	builtImages map[string]string

	FakeLogger *fakelog.FakeLogger
}

// GetLog implements interface
func (c *customFactory) GetLog() log.Logger {
	return c.FakeLogger
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

var availableSubTests = map[string]func(factory *customFactory) error{
	"default": RunDefault,
	"profile": RunProfile,
	"kubectl": RunKubectl,
	"helm":    RunHelm,
}

func (r *Runner) Run(subTests []string, ns string, pwd string) error {
	// Populates the tests to run with all the available sub tests if no sub tests are specified
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
		utils.PrintTestResult("deploy", subTestName, err)
		if err != nil {
			return err
		}
	}

	return nil
}

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

	// 2. Deploy config
	err = t.deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// 3. Analyze pods
	err = utils.AnalyzePods(client, f.namespace)
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

	for start := time.Now(); time.Since(start) < time.Second*60; {
		p, _ := client.KubeClient().CoreV1().Pods(f.namespace).List(metav1.ListOptions{})

		if len(p.Items) == 0 {
			return nil
		}
	}

	p, _ := client.KubeClient().CoreV1().Pods(f.namespace).List(metav1.ListOptions{})
	return errors.Errorf("purge command failed, expected 0 pod but found %v", len(p.Items))
}
