package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/covexo/devspace/pkg/util/ignoreutil"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	noSleep        bool
}

const pullSecretName = "devspace-pull-secret"

var UpFlagsDefault = &UpCmdFlags{
	tiller:         true,
	open:           "cmd",
	initRegistry:   true,
	build:          true,
	sync:           true,
	portforwarding: true,
	noSleep:        false,
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
	cobraCmd.Flags().BoolVar(&cmd.flags.noSleep, "no-sleep", cmd.flags.noSleep, "Enable no-sleep")
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
		logutil.PrintFailMessage(fmt.Sprintf("Unable to create new kubectl client: %s", err.Error()), os.Stdout)
		log.WithError(err).Panic("Unable to create new kubectl client")
		return
	}

	if cmd.flags.build {
		mustRebuild := true
		dockerfileInfo, statErr := os.Stat(cmd.workdir + "/Dockerfile")
		var dockerfileModTime time.Time

		if statErr != nil {
			if len(cmd.privateConfig.Release.LatestImage) == 0 {
				logutil.PrintFailMessage(fmt.Sprintf("Unable to call stat on Dockerfile: %s", statErr.Error()), os.Stdout)
				log.WithError(statErr).Panic("Unable to call stat on Dockerfile")
				return
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
				logutil.PrintFailMessage(fmt.Sprintf("Config saving error: %s", privateConfigErr.Error()), os.Stdout)
				log.WithError(privateConfigErr).Panic("Config saving error")
				return
			}
		} else {
			cmd.latestImageIP = cmd.privateConfig.Release.LatestImage
		}
	}
	cmd.deployChart()

	if cmd.flags.sync {
		loadingText := logutil.NewLoadingText("Starting real-time code sync", os.Stdout)
		cmd.startSync()
		loadingText.Done()
	}

	if cmd.flags.portforwarding {
		loadingText := logutil.NewLoadingText("Starting port forwarding", os.Stdout)
		cmd.startPortForwarding()
		loadingText.Done()
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

		loadingText := logutil.NewLoadingText("Waiting for build pod to start", os.Stdout)

		for readyWaitTime > 0 {
			buildPod, _ = cmd.kubectl.Core().Pods(buildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})

			if len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready {
				buildPodReady = true
				break
			}

			time.Sleep(readyCheckInterval)
			readyWaitTime = readyWaitTime - readyCheckInterval
		}

		loadingText.Done()

		if !buildPodReady {
			log.Panic("Unable to start build pod")
		} else {
			logutil.PrintDoneMessage("Uploading files to build container", os.Stdout)

			ignoreRules, ignoreRuleErr := ignoreutil.GetIgnoreRules(cmd.workdir)

			if ignoreRuleErr != nil {
				log.WithError(ignoreRuleErr).Panic("Unable to parse .dockerignore files")
			}
			buildContainer := &buildPod.Spec.Containers[0]

			synctool.CopyToContainer(cmd.kubectl, buildPod, buildContainer, cmd.workdir, "/src", ignoreRules)

			logutil.PrintDoneMessage("Starting build process", os.Stdout)

			containerBuildPath := "/src/" + filepath.Base(cmd.workdir)

			exitChannel := make(chan error)

			stdin, stdout, stderr, execErr := kubectl.Exec(cmd.kubectl, buildPod, buildContainer.Name, []string{
				"/kaniko/executor",
				"--dockerfile=" + containerBuildPath + "/Dockerfile",
				"--context=dir://" + containerBuildPath,
				"--destination=" + cmd.latestImageHostname,
				"--insecure-skip-tls-verify",
				"--single-snapshot",
			}, false, exitChannel)
			stdin.Close()

			if execErr != nil {
				log.WithError(execErr).Panic("Failed to start image building")
			}
			lastKanikoOutput, _ := cmd.formatKanikoOutput(stdout, stderr)
			exitError := <-exitChannel

			if exitError != nil {
				log.WithError(exitError).Panic("Image building failed with message: " + lastKanikoOutput)
			}
			logutil.PrintDoneMessage("Done building image", os.Stdout)
		}
		return nil
	})
}

type KanikoOutputFormat struct {
	Regex       *regexp.Regexp
	Replacement string
}

func (cmd *UpCmd) formatKanikoOutput(stdout io.ReadCloser, stderr io.ReadCloser) (string, *logutil.LoadingText) {
	wg := &sync.WaitGroup{}
	lastLine := ""
	outputFormats := []KanikoOutputFormat{
		{
			Regex:       regexp.MustCompile(`.* msg="Downloading base image (.*)"`),
			Replacement: " FROM $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="(Unpacking layer: \d+)"`),
			Replacement: ">> $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="cmd: Add \[(.*)\]"`),
			Replacement: " ADD $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="cmd: copy \[(.*)\]"`),
			Replacement: " COPY $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="dest: (.*)"`),
			Replacement: ">> destination: $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="args: \[-c (.*)\]"`),
			Replacement: " RUN $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Replacing CMD in config with \[(.*)\]"`),
			Replacement: " CMD $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Changed working directory to (.*)"`),
			Replacement: " WORKDIR $1",
		},
		{
			Regex:       regexp.MustCompile(`.* msg="Taking snapshot of full filesystem..."`),
			Replacement: " Packaging layers",
		},
	}
	var latestLoadingText *logutil.LoadingText
	kanikoLogRegex := regexp.MustCompile(`^time="(.*)" level=(.*) msg="(.*)"`)
	buildPrefix := "build >"

	printFormattedOutput := func(originalLine string) {
		line := []byte(originalLine)

		for _, outputFormat := range outputFormats {
			line = outputFormat.Regex.ReplaceAll(line, []byte(outputFormat.Replacement))
		}
		lineString := string(line)

		if len(line) != len(originalLine) {
			if latestLoadingText != nil {
				latestLoadingText.Done()
			}
			latestLoadingText = logutil.NewLoadingText(buildPrefix+lineString, os.Stdout)
		} else if kanikoLogRegex.Match(line) == false {
			if latestLoadingText != nil {
				latestLoadingText.Done()
			}
			log.Info(buildPrefix + ">> " + lineString)
		}
		lastLine = string(kanikoLogRegex.ReplaceAll([]byte(originalLine), []byte("$3")))
	}
	processutil.RunOnEveryLine(stdout, printFormattedOutput, 500, wg)
	processutil.RunOnEveryLine(stderr, printFormattedOutput, 500, wg)

	wg.Wait()

	return lastLine, latestLoadingText
}

func (cmd *UpCmd) initRegistry() {
	loadingText := logutil.NewLoadingText("Initializing helm client", os.Stdout)
	cmd.initHelm()
	loadingText.Done()

	installRegistry := cmd.flags.initRegistry

	if installRegistry {
		loadingText = logutil.NewLoadingText("Initializing docker registry", os.Stdout)

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
		_, deploymentErr := cmd.helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", &registryConfig)

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

		loadingText.Done()
		loadingText = logutil.NewLoadingText("Waiting for registry service to start", os.Stdout)

		for true {
			registryService, _ = cmd.kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

			if len(registryService.Spec.ClusterIP) > 0 {
				break
			}

			time.Sleep(serviceWaitingInterval)
			maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

			if maxServiceWaiting <= 0 {
				log.Panic("Timeout waiting for registry service to start")
			}
		}
		loadingText.Done()

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
	logutil.PrintDoneMessage("Deploying helm chart", os.Stdout)
	cmd.initHelm()

	releaseName := cmd.privateConfig.Release.Name
	releaseNamespace := cmd.privateConfig.Release.Namespace
	chartPath := "chart/"

	values := map[interface{}]interface{}{}
	containerValues := map[interface{}]interface{}{}

	containerValues["image"] = cmd.latestImageIP
	if !cmd.flags.noSleep {
		containerValues["command"] = []string{"sleep"}
		containerValues["args"] = []string{"99999999"}
	}
	values["container"] = containerValues

	appRelease, deploymentErr := cmd.helm.InstallChartByPath(releaseName, releaseNamespace, chartPath, &values)

	if deploymentErr != nil {
		log.WithError(deploymentErr).Panic("Unable to deploy helm chart")
	}
	releaseRevision := int(appRelease.Version)

	for true {
		podList, podListErr := cmd.kubectl.Core().Pods(releaseNamespace).List(metav1.ListOptions{
			LabelSelector: "release=" + releaseName,
		})

		if podListErr != nil {
			log.WithError(podListErr).Panic("Unable to list devspace pods")
		}

		if len(podList.Items) > 0 {
			highestRevision := 0
			var selectedPod k8sv1.Pod

			for i, pod := range podList.Items {
				podRevision, podHasRevision := pod.Labels["revision"]
				hasHigherRevision := (i == 0)

				if !hasHigherRevision && podHasRevision {
					podRevisionInt, _ := strconv.Atoi(podRevision)

					if podRevisionInt > highestRevision {
						hasHigherRevision = true
					}
				}

				if hasHigherRevision {
					selectedPod = pod
					highestRevision, _ = strconv.Atoi(podRevision)
				}
			}
			_, hasRevision := selectedPod.Labels["revision"]

			if !hasRevision || highestRevision == releaseRevision {
				if !hasRevision {
					log.Warn("Found pod without revision. Use label 'revision' for your pods to avoid this warning.")
				}
				cmd.pod = &selectedPod

				waitForPodReady(cmd.kubectl, cmd.pod, 2*60*time.Second, 5*time.Second, "Waiting for DevSpace pod to become ready")
				break
			} else {
				log.Info("Waiting for release upgrade to complete.")
			}
		} else {
			log.Info("Waiting for release to be deployed.")
		}
		time.Sleep(2 * time.Second)
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

func (cmd *UpCmd) startPortForwarding() {
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
						case <-time.After(5 * time.Second):
							log.Error("Timeout waiting for port forwarding to start")
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
	log.Info("Starting terminal session")

	_, _, _, terminalErr := kubectl.Exec(cmd.kubectl, cmd.pod, cmd.pod.Spec.Containers[0].Name, shell, true, nil)

	if terminalErr != nil {
		if _, ok := terminalErr.(exec.CodeExitError); ok == false {
			log.WithError(terminalErr).Panic("Unable to start terminal session")
		}
	}
}

func waitForPodReady(kubectl *kubernetes.Clientset, pod *k8sv1.Pod, maxWaitTime time.Duration, checkInterval time.Duration, waitingMessage string) error {
	loadingText := logutil.NewLoadingText(waitingMessage, os.Stdout)
	defer loadingText.Done()

	for maxWaitTime > 0 {
		pod, _ := kubectl.Core().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})

		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		}

		time.Sleep(checkInterval)
		maxWaitTime = maxWaitTime - checkInterval
	}
	return errors.New("")
}
