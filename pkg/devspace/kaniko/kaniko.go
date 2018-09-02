package kaniko

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	synctool "github.com/covexo/devspace/pkg/devspace/sync"
	"github.com/covexo/devspace/pkg/util/ignoreutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/util/interrupt"
)

// BuildDockerfile builds a dockerfile in a kaniko build pod
func BuildDockerfile(client *kubernetes.Clientset, buildNamespace, imageDestination, pullSecretName string) error {
	//registrySecretName := cmd.privateConfig.Registry.Release.Name + "-docker-registry-secret"
	//registryHostname := cmd.privateConfig.Registry.Release.Name + "-docker-registry." + cmd.privateConfig.Registry.Release.Namespace + ".svc.cluster.local:5000"
	workdir, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("Unable to determine current workdir: %s", err.Error())
	}

	randString, _ := randutil.GenerateRandomString(12)
	buildID := strings.ToLower(randString)
	buildPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "devspace-build-",
			Labels: map[string]string{
				"devspace-build-id": buildID,
			},
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:            "kaniko",
					Image:           "gcr.io/kaniko-project/executor:debug-60bdda4c49b699f5a21545cc8a050a5f3953223f",
					ImagePullPolicy: k8sv1.PullIfNotPresent,
					Command: []string{
						"/busybox/sleep",
					},
					Args: []string{
						"36000",
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:      pullSecretName,
							MountPath: "/root/.docker",
						},
					},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: pullSecretName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: pullSecretName,
							Items: []k8sv1.KeyToPath{
								{
									Key:  k8sv1.DockerConfigJsonKey,
									Path: "config.json",
								},
							},
						},
					},
				},
			},
			RestartPolicy: k8sv1.RestartPolicyOnFailure,
		},
	}

	deleteBuildPod := func() {
		gracePeriod := int64(3)

		deleteErr := client.Core().Pods(buildNamespace).Delete(buildPod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			log.Errorf("Failed to delete build pod: %s", deleteErr.Error())
		}
	}

	intr := interrupt.New(nil, deleteBuildPod)

	err = intr.Run(func() error {
		buildPodCreated, buildPodCreateErr := client.Core().Pods(buildNamespace).Create(buildPod)

		if buildPodCreateErr != nil {
			return fmt.Errorf("Unable to create build pod: %s", buildPodCreateErr.Error())
		}

		readyWaitTime := 2 * 60 * time.Second
		readyCheckInterval := 5 * time.Second
		buildPodReady := false

		log.StartWait("Waiting for kaniko build pod to start")

		for readyWaitTime > 0 {
			buildPod, _ = client.Core().Pods(buildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})

			if len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready {
				buildPodReady = true
				break
			}

			time.Sleep(readyCheckInterval)
			readyWaitTime = readyWaitTime - readyCheckInterval
		}

		log.StopWait()
		log.Done("Kaniko build pod started")

		if !buildPodReady {
			return fmt.Errorf("Unable to start build pod")
		}
		ignoreRules, ignoreRuleErr := ignoreutil.GetIgnoreRules(workdir)

		if ignoreRuleErr != nil {
			return fmt.Errorf("Unable to parse .dockerignore files: %s", ignoreRuleErr.Error())
		}

		buildContainer := &buildPod.Spec.Containers[0]

		log.StartWait("Uploading files to build container")
		err := synctool.CopyToContainer(client, buildPod, buildContainer, workdir, "/src", ignoreRules)
		log.StopWait()

		if err != nil {
			return fmt.Errorf("Error uploading files to container: %s", err.Error())
		}

		log.Done("Uploaded files to container")
		log.StartWait("Building container image")

		containerBuildPath := "/src/" + filepath.Base(workdir)
		exitChannel := make(chan error)

		stdin, stdout, stderr, execErr := kubectl.Exec(client, buildPod, buildContainer.Name, []string{
			"/kaniko/executor",
			"--dockerfile=" + containerBuildPath + "/Dockerfile",
			"--context=dir://" + containerBuildPath,
			"--destination=" + imageDestination,
			"--insecure-skip-tls-verify",
			"--single-snapshot",
		}, false, exitChannel)

		stdin.Close()

		if execErr != nil {
			return fmt.Errorf("Failed to start image building: %s", execErr.Error())
		}

		lastKanikoOutput := formatKanikoOutput(stdout, stderr)
		exitError := <-exitChannel

		log.StopWait()

		if exitError != nil {
			return fmt.Errorf("Error: %s, Last Kaniko Output: %s", exitError.Error(), lastKanikoOutput)
		}

		log.Done("Done building image")

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
