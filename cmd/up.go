package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/logutil"

	"k8s.io/kubernetes/pkg/util/interrupt"

	synctool "github.com/covexo/devspace/pkg/devspace/sync"

	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/util/processutil"
	"github.com/covexo/devspace/pkg/util/randutil"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	glob "github.com/bmatcuk/doublestar"
	"github.com/foomo/htpasswd"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/exec"
)

type UpCmd struct {
	flags               *UpCmdFlags
	helm                *helmClient.HelmClientWrapper
	kubectl             *kubernetes.Clientset
	privateConfig       *v1.PrivateConfig
	dsConfig            *v1.DevSpaceConfig
	workdir             string
	pod                 *k8sv1.Pod
	container           *k8sv1.Container
	latestImageHostname string
	latestImageIP       string
}

type UpCmdFlags struct {
	tiller         bool
	open           string
	initRegistry   bool
	build          bool
	shell          string
	sync           bool
	portforwarding bool
}

const pullSecretName = "devspace-pull-secret"

var UpFlagsDefault = &UpCmdFlags{
	tiller:         true,
	open:           "cmd",
	initRegistry:   true,
	build:          true,
	sync:           true,
	portforwarding: true,
}

func init() {
	cmd := &UpCmd{
		flags: UpFlagsDefault,
	}

	cobraCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts your DevSpace",
		Long: `
#######################################################
#################### devspace up ######################
#######################################################
Starts and connects your DevSpace:
1. Connects to the Tiller server
2. Builds your Docker image (if your Dockerfile has changed)
3. Deploys the Helm chart in /chart
4. Starts the sync client
5. Enters the container shell
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVar(&cmd.flags.tiller, "tiller", cmd.flags.tiller, "Install/upgrade tiller")
	cobraCmd.Flags().StringVarP(&cmd.flags.open, "open", "o", cmd.flags.open, "Install/upgrade tiller")
	cobraCmd.Flags().BoolVar(&cmd.flags.initRegistry, "init-registry", cmd.flags.initRegistry, "Install or upgrade Docker registry")
	cobraCmd.Flags().BoolVarP(&cmd.flags.build, "build", "b", cmd.flags.build, "Build image if Dockerfile has been modified")
	cobraCmd.Flags().StringVarP(&cmd.flags.shell, "shell", "s", "", "Shell command (default: bash, fallback: sh)")
	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")
}

func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	log = logutil.GetLogger("default", true)
	var err error
	workdir, workdirErr := os.Getwd()

	if workdirErr != nil {
		log.WithError(workdirErr).Panic("Unable to determine current workdir")
	}
	cmd.workdir = workdir
	cmd.privateConfig = &v1.PrivateConfig{}
	cmd.dsConfig = &v1.DevSpaceConfig{}

	privateConfigExists, _ := config.ConfigExists(cmd.privateConfig)
	dsConfigExists, _ := config.ConfigExists(cmd.dsConfig)

	if !privateConfigExists || !dsConfigExists {
		initCmd := &InitCmd{
			flags: InitCmdFlagsDefault,
		}
		initCmd.Run(nil, []string{})
	}
	config.LoadConfig(cmd.privateConfig)
	config.LoadConfig(cmd.dsConfig)

	cmd.kubectl, err = kubectl.NewClient()

	if err != nil {
		log.WithError(err).Panic("Unable to create new kubectl client")
	}

	if cmd.flags.build {
		mustRebuild := true
		dockerfileInfo, statErr := os.Stat(cmd.workdir + "/Dockerfile")
		var dockerfileModTime time.Time

		if statErr != nil {
			if len(cmd.privateConfig.Release.LatestImage) == 0 {
				log.WithError(statErr).Panic("Unable to call stat on Dockerfile")
			} else {
				mustRebuild = false
			}
		} else {
			dockerfileModTime = dockerfileInfo.ModTime()

			// When user has not used -b or --build flags
			if cobraCmd.Flags().Changed("build") == false {
				if len(cmd.privateConfig.Release.LatestBuild) != 0 {
					latestBuildTime, _ := time.Parse(time.RFC3339Nano, cmd.privateConfig.Release.LatestBuild)

					// only rebuild Docker image when Dockerfile has changed since latest build
					mustRebuild = (latestBuildTime.Equal(dockerfileModTime) == false)
				}
			}
		}

		if mustRebuild {
			cmd.buildDockerfile()

			cmd.privateConfig.Release.LatestBuild = dockerfileModTime.Format(time.RFC3339Nano)
			cmd.privateConfig.Release.LatestImage = cmd.latestImageIP

			privateConfigErr := config.SaveConfig(cmd.privateConfig)

			if privateConfigErr != nil {
				log.WithError(privateConfigErr).Panic("Config saving error")
			}
		} else {
			cmd.latestImageIP = cmd.privateConfig.Release.LatestImage
		}
	}
	cmd.deployChart()

	if cmd.flags.sync {
		cmd.startSync()
	}

	if cmd.flags.portforwarding {
		cmd.startPortForwards()
	}
	cmd.enterTerminal()
}

func (cmd *UpCmd) buildDockerfile() {
	cmd.initRegistry()

	//registrySecretName := cmd.privateConfig.Registry.Release.Name + "-docker-registry-secret"
	//registryHostname := cmd.privateConfig.Registry.Release.Name + "-docker-registry." + cmd.privateConfig.Registry.Release.Namespace + ".svc.cluster.local:5000"
	buildNamespace := cmd.privateConfig.Release.Namespace
	randString, _ := randutil.GenerateRandomString(12)
	buildId := strings.ToLower(randString)
	buildPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "devspace-build-",
			Labels: map[string]string{
				"devspace-build-id": buildId,
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
						k8sv1.VolumeMount{
							Name:      pullSecretName,
							MountPath: "/root/.docker",
						},
					},
				},
			},
			Volumes: []k8sv1.Volume{
				k8sv1.Volume{
					Name: pullSecretName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: pullSecretName,
							Items: []k8sv1.KeyToPath{
								k8sv1.KeyToPath{
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

		deleteErr := cmd.kubectl.Core().Pods(buildNamespace).Delete(buildPod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			log.WithError(deleteErr).Error("Failed delete build pod")
		}
	}
	intr := interrupt.New(nil, deleteBuildPod)

	intr.Run(func() error {
		buildPodCreated, buildPodCreateErr := cmd.kubectl.Core().Pods(buildNamespace).Create(buildPod)

		if buildPodCreateErr != nil {
			log.WithError(buildPodCreateErr).Panic("Unable to create build pod")
		}

		readyWaitTime := 2 * 60 * time.Second
		readyCheckInterval := 5 * time.Second
		buildPodReady := false

		for readyWaitTime > 0 {
			buildPod, _ = cmd.kubectl.Core().Pods(buildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})

			if len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready {
				buildPodReady = true
				break
			}
			log.Info("Waiting for build pod to start")

			time.Sleep(readyCheckInterval)

			readyWaitTime = readyWaitTime - readyCheckInterval
		}

		if !buildPodReady {
			log.Panic("Unable to start build pod")
		} else {
			log.Info("Uploading files to build container")

			ignoreRules := []string{}
			ignoreFiles, err := glob.Glob(cmd.workdir + "/**/.dockerignore")

			if err != nil {
				return err
			}

			for _, ignoreFile := range ignoreFiles {
				fmt.Println(ignoreFile)
				ignoreBytes, err := ioutil.ReadFile(ignoreFile)

				if err != nil {
					return err
				}
				pathPrefix := strings.Replace(strings.TrimPrefix(filepath.Dir(ignoreFile), cmd.workdir), "\\", "/", -1)
				ignoreLines := strings.Split(string(ignoreBytes), "\r\n")

				for _, ignoreRule := range ignoreLines {
					ignoreRule = strings.Trim(ignoreRule, " ")
					initialOffset := 0

					if len(ignoreRule) > 0 && ignoreRule[initialOffset] != '#' {
						prefixedIgnoreRule := ""

						if len(pathPrefix) > 0 {
							if ignoreRule[initialOffset] == '!' {
								prefixedIgnoreRule = prefixedIgnoreRule + "!"
								initialOffset = 1
							}

							if ignoreRule[initialOffset] == '/' {
								prefixedIgnoreRule = prefixedIgnoreRule + ignoreRule[initialOffset:]
							} else {
								prefixedIgnoreRule = prefixedIgnoreRule + pathPrefix + "/**/" + ignoreRule[initialOffset:]
							}
						} else {
							prefixedIgnoreRule = ignoreRule
						}

						if prefixedIgnoreRule != "Dockerfile" && prefixedIgnoreRule != "/Dockerfile" {
							ignoreRules = append(ignoreRules, prefixedIgnoreRule)
						}
					}
				}
			}
			buildContainer := &buildPod.Spec.Containers[0]

			synctool.CopyToContainer(cmd.kubectl, buildPod, buildContainer, cmd.workdir, "/src", ignoreRules)

			log.Info("Starting build process")

			containerBuildPath := "/src/" + filepath.Base(cmd.workdir)

			stdin, stdout, stderr, execErr := kubectl.Exec(cmd.kubectl, buildPod, buildContainer.Name, []string{
				"/kaniko/executor",
				"--dockerfile=" + containerBuildPath + "/Dockerfile",
				"--context=dir://" + containerBuildPath,
				"--destination=" + cmd.latestImageHostname,
				"--insecure-skip-tls-verify",
				"--single-snapshot",
			}, false)
			stdin.Close()

			wg := &sync.WaitGroup{}

			//TODO: use logger?
			processutil.Pipe(stdout, os.Stdout, 500, wg)
			processutil.Pipe(stderr, os.Stderr, 500, wg)

			wg.Wait()

			if execErr != nil {
				log.WithError(execErr).Panic("Failed building image")
			}
			log.Info("Done building image")
		}
		return nil
	})
}

func (cmd *UpCmd) initRegistry() {
	log.Info("Initializing helm client")
	cmd.initHelm()

	installRegistry := cmd.flags.initRegistry

	if installRegistry {
		registryReleaseName := cmd.privateConfig.Registry.Release.Name
		registryReleaseNamespace := cmd.privateConfig.Registry.Release.Namespace
		registryConfig := cmd.dsConfig.Registry
		registrySecrets, secretsExist := registryConfig["secrets"]

		if !secretsExist {
			//TODO
		}
		_, secretIsMap := registrySecrets.(map[interface{}]interface{})

		if !secretIsMap {
			//TODO
		}
		log.Info("Initializing docker registry")

		deploymentErr := cmd.helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", &registryConfig)

		if deploymentErr != nil {
			log.WithError(deploymentErr).Panic("Unable to initialize docker registry")
		}
		htpasswdSecretName := registryReleaseName + "-docker-registry-secret"
		htpasswdSecret, secretGetErr := cmd.kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		if secretGetErr != nil {
			log.WithError(secretGetErr).Panic("Unable to retrieve secret for docker registry")
		}

		if htpasswdSecret == nil || htpasswdSecret.Data == nil {
			htpasswdSecret = &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: htpasswdSecretName,
				},
				Data: map[string][]byte{},
			}
		}
		oldHtpasswdData := htpasswdSecret.Data["htpasswd"]
		newHtpasswdData := htpasswd.HashedPasswords{}

		if len(oldHtpasswdData) != 0 {
			oldHtpasswdDataBytes := []byte(oldHtpasswdData)
			newHtpasswdData, _ = htpasswd.ParseHtpasswd(oldHtpasswdDataBytes)
		}
		registryUser := cmd.privateConfig.Registry.User
		htpasswdErr := newHtpasswdData.SetPassword(registryUser.Username, registryUser.Password, htpasswd.HashBCrypt)

		if htpasswdErr != nil {
			log.WithError(htpasswdErr).Panic("Unable to set password in htpasswd")
		}
		newHtpasswdDataBytes := newHtpasswdData.Bytes()

		htpasswdSecret.Data["htpasswd"] = newHtpasswdDataBytes

		_, getHtpasswdSecretErr := cmd.kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		var htpasswdUpdateErr error

		if getHtpasswdSecretErr != nil {
			_, htpasswdUpdateErr = cmd.kubectl.Core().Secrets(registryReleaseNamespace).Create(htpasswdSecret)
		} else {
			_, htpasswdUpdateErr = cmd.kubectl.Core().Secrets(registryReleaseNamespace).Update(htpasswdSecret)
		}

		if htpasswdUpdateErr != nil {
			log.WithError(htpasswdUpdateErr).Panic("Unable to update htpasswd secret")
		}
		registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(cmd.privateConfig.Registry.User.Username + ":" + cmd.privateConfig.Registry.User.Password))

		registryServiceName := registryReleaseName + "-docker-registry"
		var registryService *k8sv1.Service
		maxServiceWaiting := 60 * time.Second
		serviceWaitingInterval := 3 * time.Second

		for true {
			registryService, _ = cmd.kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

			if len(registryService.Spec.ClusterIP) > 0 {
				break
			}
			log.Info("Waiting for registry service to start")
			time.Sleep(serviceWaitingInterval)
			maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

			if maxServiceWaiting <= 0 {
				log.Panic("Timeout waiting for registry service to start")
			}
		}
		registryPort := 5000
		registryIP := registryService.Spec.ClusterIP + ":" + strconv.Itoa(registryPort)
		registryHostname := registryServiceName + "." + registryReleaseNamespace + ".svc.cluster.local:" + strconv.Itoa(registryPort)
		latestImageTag, _ := randutil.GenerateRandomString(10)

		cmd.latestImageHostname = registryHostname + "/" + cmd.privateConfig.Release.Name + ":" + latestImageTag
		cmd.latestImageIP = registryIP + "/" + cmd.privateConfig.Release.Name + ":" + latestImageTag

		pullSecretDataValue := []byte(`{
			"auths": {
				"` + registryHostname + `": {
					"auth": "` + registryAuthEncoded + `",
					"email": "noreply-devspace@covexo.com"
				},
				
				"` + registryIP + `": {
					"auth": "` + registryAuthEncoded + `",
					"email": "noreply-devspace@covexo.com"
				}
			}
		}`)

		pullSecretData := map[string][]byte{}
		pullSecretDataKey := k8sv1.DockerConfigJsonKey
		pullSecretData[pullSecretDataKey] = pullSecretDataValue

		registryPullSecret := &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: pullSecretName,
			},
			Data: pullSecretData,
			Type: k8sv1.SecretTypeDockerConfigJson,
		}

		_, getImagePullSecretErr := cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Get(pullSecretName, metav1.GetOptions{})

		var secretCreationErr error

		if getImagePullSecretErr != nil {
			_, secretCreationErr = cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Create(registryPullSecret)
		} else {
			_, secretCreationErr = cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Update(registryPullSecret)
		}

		if secretCreationErr != nil {
			log.WithError(secretCreationErr).Panic("Unable to update image pull secret")
		}
	}
}

func (cmd *UpCmd) initHelm() {
	if cmd.helm == nil {
		client, helmErr := helmClient.NewClient(cmd.kubectl, false)

		if helmErr != nil {
			log.WithError(helmErr).Panic("Unable to initialize helm client")
		}
		cmd.helm = client
	}
}

func (cmd *UpCmd) deployChart() {
	log.Info("Deploying helm chart")
	cmd.initHelm()

	releaseName := cmd.privateConfig.Release.Name
	releaseNamespace := cmd.privateConfig.Release.Namespace
	chartPath := "chart/"

	values := map[interface{}]interface{}{}
	containerValues := map[interface{}]interface{}{}
	containerValues["image"] = cmd.latestImageIP
	containerValues["command"] = []string{"sleep", "99999999"}
	values["container"] = containerValues

	deploymentErr := cmd.helm.InstallChartByPath(releaseName, releaseNamespace, chartPath, &values)

	if deploymentErr != nil {
		log.WithError(deploymentErr).Panic("Unable to deploy helm chart")
	}

	for true {
		podList, podListErr := cmd.kubectl.Core().Pods(releaseNamespace).List(metav1.ListOptions{
			LabelSelector: "release=" + releaseName,
		})

		if podListErr != nil {
			log.WithError(podListErr).Panic("Unable to list devspace pods")
		}

		if len(podList.Items) > 0 {
			cmd.pod = &podList.Items[0]

			waitForPodReady(cmd.kubectl, cmd.pod, 2*60*time.Second, 10*time.Second, "Waiting for DevSpace pod to become ready")
			break
		}
	}
}

func (cmd *UpCmd) startSync() {
	for _, syncPath := range cmd.dsConfig.SyncPaths {
		absLocalPath, err := filepath.Abs(cmd.workdir + syncPath.LocalSubPath)

		if err != nil {
			log.WithError(err).Panic("Unable to resolve localSubPath: " + syncPath.LocalSubPath)
		} else {
			syncConfig := synctool.SyncConfig{
				Kubectl:   cmd.kubectl,
				Pod:       cmd.pod,
				Container: &cmd.pod.Spec.Containers[0],
				WatchPath: absLocalPath,
				DestPath:  syncPath.ContainerPath,
			}
			syncConfig.Start()
		}
	}
}

func (cmd *UpCmd) startPortForwards() {
	for _, portForwarding := range cmd.dsConfig.PortForwarding {
		if portForwarding.ResourceType == "pod" {
			if len(portForwarding.LabelSelector) > 0 {
				labels := make([]string, 0, len(portForwarding.LabelSelector))

				for key, value := range portForwarding.LabelSelector {
					labels = append(labels, key+"="+value)
				}

				podList, podListErr := cmd.kubectl.Core().Pods(cmd.privateConfig.Registry.Release.Namespace).List(metav1.ListOptions{
					LabelSelector: strings.Join(labels, ", "),
				})

				if podListErr != nil {
					log.WithError(podListErr).Error("Unable to list devspace pods")
				} else {
					if len(podList.Items) > 0 {
						ports := make([]string, len(portForwarding.PortMappings))

						for index, value := range portForwarding.PortMappings {
							ports[index] = strconv.Itoa(value.LocalPort) + ":" + strconv.Itoa(value.RemotePort)
						}
						readyChan := make(chan struct{})

						go kubectl.ForwardPorts(cmd.kubectl, &podList.Items[0], ports, make(chan struct{}), readyChan)

						// Wait till forwarding is ready
						select {
						case <-readyChan:
						}
					}
				}
			}
		} else {
			log.Warn("Currently only pod resource type is supported for portforwarding")
		}
	}
}

func (cmd *UpCmd) enterTerminal() {
	var shell []string

	if len(cmd.flags.shell) == 0 {
		shell = []string{
			"sh",
			"-c",
			"command -v bash >/dev/null 2>&1 && exec bash || exec sh",
		}
	} else {
		shell = []string{cmd.flags.shell}
	}
	_, _, _, terminalErr := kubectl.Exec(cmd.kubectl, cmd.pod, cmd.pod.Spec.Containers[0].Name, shell, true)

	if terminalErr != nil {
		if _, ok := terminalErr.(exec.CodeExitError); ok == false {
			log.WithError(terminalErr).Panic("Unable to start terminal session")
		}
	}
}

func waitForPodReady(kubectl *kubernetes.Clientset, pod *k8sv1.Pod, maxWaitTime time.Duration, checkInterval time.Duration, waitingMessage string) error {
	for maxWaitTime > 0 {
		pod, _ := kubectl.Core().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})

		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		}
		log.Info(waitingMessage)
		time.Sleep(checkInterval)

		maxWaitTime = maxWaitTime - checkInterval
	}
	return errors.New("")
}
