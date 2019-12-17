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

func runDefault(f *customFactory) error {
	log.GetInstance().Info("Run test 'default' of 'enter'")

	client, err := f.NewKubeClientFromContext("", f.namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	pods, err := client.KubeClient().CoreV1().Pods(f.namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	podName := pods.Items[0].Name

	enterConfigs := []*cmd.EnterCmd{
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait:      true,
			Container: "container-0",
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait: true,
			Pod:  podName,
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait:          true,
			LabelSelector: "app.kubernetes.io/component=quickstart",
		},
		{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.namespace,
				NoWarn:    true,
				Silent:    true,
			},
			Wait: true,
			Pick: true,
		},
	}

	for _, c := range enterConfigs {
		done := utils.Capture()

		output := "testblabla"
		err = c.Run(f, nil, []string{"echo", output})
		if err != nil {
			return err
		}

		time.Sleep(time.Second * 5)

		capturedOutput, err := done()
		if err != nil {
			return err
		}
		fmt.Println(capturedOutput)

		if !strings.HasPrefix(capturedOutput, output) {
			return errors.New("capturedOutput is different than output for the enter cmd")
		}
	}

	return nil
}
