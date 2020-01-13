package enter

import (
	"fmt"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//1. enter --container
//2. enter --pod
//3. enter --label-selector
//4. enter --pick

func runDefault(f *utils.BaseCustomFactory, logger log.Logger) error {
	logger.Info("Run sub test 'default' of test 'enter'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	client, err := f.NewKubeClientFromContext("", f.Namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	pods, err := client.KubeClient().CoreV1().Pods(f.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Errorf("Unable to list the pods: %v", err)
	}

	podName := pods.Items[0].Name

	enterConfigs := []*cmd.EnterCmd{
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait:      true,
			Container: "container-0",
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait: true,
			Pod:  podName,
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait:          true,
			LabelSelector: "app=test",
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait: true,
			Pick: true,
		},
	}

	for _, c := range enterConfigs {
		fmt.Printf("CONFIG TO RUN: %+v", c)
		done := utils.Capture()

		output := "My Test Data"
		err = c.Run(f, nil, []string{"echo", output})
		if err != nil {
			return err
		}

		time.Sleep(time.Second * 5)

		capturedOutput, err := done()
		if err != nil {
			return err
		}

		if !strings.HasPrefix(capturedOutput, output) {
			return errors.Errorf("capturedOutput '%s' is different than output '%s' for the enter cmd", capturedOutput, output)
		}
	}

	return nil
}
