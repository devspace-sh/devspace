package examples

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/cmd/use"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunProfiles runs the test for the kustomize example
func RunProfiles(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = log.NewStreamLogger(buff, logrus.InfoLevel)

	logger.Info("Run sub test 'profiles' of test 'examples'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	var deployConfig = &cmd.DeployCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.namespace,
			NoWarn:    true,
		},
		ForceBuild:  true,
		ForceDeploy: true,
		SkipPush:    true,
	}

	err := utils.ChangeWorkingDir(f.pwd+"/../examples/profiles", f.cacheLogger)
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(deployConfig.KubeContext, deployConfig.Namespace, deployConfig.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// At last, we delete the current namespace
	defer utils.DeleteNamespace(client, deployConfig.Namespace)

	err = runProfile(f, deployConfig, "dev-service2-only", client, f.namespace, []string{"service-2"}, false)
	if err != nil {
		return err
	}

	err = runProfile(f, deployConfig, "", client, f.namespace, []string{"service-1", "service-2"}, true)
	if err != nil {
		return err
	}

	return nil
}

func runProfile(f *customFactory, deployConfig *cmd.DeployCmd, profile string, client kubectl.Client, namespace string, expectedPodLabels []string, reset bool) error {
	var profileConfig = &use.ProfileCmd{
		Reset: reset,
	}

	if profile == "" {
		err := profileConfig.RunUseProfile(f, nil, nil)
		if err != nil {
			return err
		}
	} else {
		err := profileConfig.RunUseProfile(f, nil, []string{profile})
		if err != nil {
			return err
		}
	}

	err := profileConfig.RunUseProfile(f, nil, []string{profile})
	if err != nil {
		return err
	}

	err = deployConfig.Run(f, nil, nil)
	if err != nil {
		return err
	}

	// Checking if pods are running correctly
	err = utils.AnalyzePods(client, f.namespace, f.cacheLogger)
	if err != nil {
		return err
	}

	pods, errp := client.KubeClient().CoreV1().Pods(namespace).List(v1.ListOptions{})
	if errp != nil {
		return err
	}

	var rp []string
	for _, x := range expectedPodLabels {
		for _, y := range pods.Items {
			if x == y.ObjectMeta.Labels["app.kubernetes.io/component"] {
				rp = append(rp, x)
			}
		}
	}

	if !utils.Equal(rp, expectedPodLabels) {
		return errors.New("The expected pods are not running")
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
	err = utils.PortForwardAndPing(config, generatedConfig, client, f.cacheLogger)
	if err != nil {
		return err
	}

	return nil
}
