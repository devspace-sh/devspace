package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/ignoreutil"
	"github.com/covexo/devspace/pkg/util/log"

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
	tiller           bool
	open             string
	initRegistry     bool
	build            bool
	shell            string
	sync             bool
	portforwarding   bool
	noSleep          bool
	imageDestination string
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
	cobraCmd.Flags().StringVarP(&cmd.flags.imageDestination, "image-destination", "", "", "Choose image destination")
}

// Run executes the command logic
func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	workdir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
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

	err = config.LoadConfig(cmd.privateConfig)

	if err != nil {
		log.Fatalf("Couldn't load private.yaml: %s", err.Error())
	}

	err = config.LoadConfig(cmd.dsConfig)

	if err != nil {
		log.Fatalf("Couldn't load config.yaml: %s", err.Error())
	}

	cmd.kubectl, err = kubectl.NewClient()

	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %s", err.Error())
	}

	if cmd.flags.build {
		mustRebuild := true
		dockerfileInfo, statErr := os.Stat(cmd.workdir + "/Dockerfile")
		var dockerfileModTime time.Time

		if statErr != nil {
			if len(cmd.privateConfig.Release.LatestImage) == 0 {
				log.Fatalf("Dockerfile missing: %s", statErr.Error())
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

			err = config.SaveConfig(cmd.privateConfig)

			if err != nil {
				log.Fatalf("Config saving error: %s", err.Error())
			}
		} else {
			cmd.latestImageIP = cmd.privateConfig.Release.LatestImage
		}
	}

	cmd.deployChart()

	if cmd.flags.sync {
		log.StartWait("Starting real-time code sync")
		cmd.startSync()
		log.StopWait()
	}

	if cmd.flags.portforwarding {
		log.StartWait("Starting port forwarding")
		cmd.startPortForwarding()
		log.StopWait()
	}

	cmd.enterTerminal()
}

func (cmd *UpCmd) buildDockerfile() {
	cmd.initRegistry()

	//registrySecretName := cmd.privateConfig.Registry.Release.Name + "-docker-registry-secret"
	//registryHostname := cmd.privateConfig.Registry.Release.Name + "-docker-registry." + cmd.privateConfig.Registry.Release.Namespace + ".svc.cluster.local:5000"
	buildNamespace := cmd.privateConfig.Release.Namespace
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

		deleteErr := cmd.kubectl.Core().Pods(buildNamespace).Delete(buildPod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			log.With(deleteErr).Errorf("Failed to delete build pod: %s", deleteErr.Error())
		}
	}

	intr := interrupt.New(nil, deleteBuildPod)

	err := intr.Run(func() error {
		buildPodCreated, buildPodCreateErr := cmd.kubectl.Core().Pods(buildNamespace).Create(buildPod)

		if buildPodCreateErr != nil {
			return fmt.Errorf("Unable to create build pod: %s", buildPodCreateErr.Error())
		}

		readyWaitTime := 2 * 60 * time.Second
		readyCheckInterval := 5 * time.Second
		buildPodReady := false

		log.StartWait("Waiting for build pod to start")

		for readyWaitTime > 0 {
			buildPod, _ = cmd.kubectl.Core().Pods(buildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})

			if len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready {
				buildPodReady = true
				break
			}

			time.Sleep(readyCheckInterval)
			readyWaitTime = readyWaitTime - readyCheckInterval
		}

		log.StopWait()
		log.Done("Build pod started")

		if !buildPodReady {
			return fmt.Errorf("Unable to start build pod")
		} else {
			ignoreRules, ignoreRuleErr := ignoreutil.GetIgnoreRules(cmd.workdir)

			if ignoreRuleErr != nil {
				return fmt.Errorf("Unable to parse .dockerignore files: %s", ignoreRuleErr.Error())
			}

			buildContainer := &buildPod.Spec.Containers[0]

			log.StartWait("Uploading files to build container")
			err := synctool.CopyToContainer(cmd.kubectl, buildPod, buildContainer, cmd.workdir, "/src", ignoreRules)
			log.StopWait()

			if err != nil {
				return fmt.Errorf("Error uploading files to container: %s", err.Error())
			}

			log.Done("Uploaded files to container")
			log.StartWait("Building container image")

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
				return fmt.Errorf("Failed to start image building: %s", execErr.Error())
			}

			lastKanikoOutput := cmd.formatKanikoOutput(stdout, stderr)
			exitError := <-exitChannel

			log.StopWait()

			if exitError != nil {
				return fmt.Errorf("Error: %s, Last Kaniko Output: %s", exitError.Error(), lastKanikoOutput)
			}

			log.Done("Done building image")
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Image building failed: %s", err.Error())
	}
}

type KanikoOutputFormat struct {
	Regex       *regexp.Regexp
	Replacement string
}

func (cmd *UpCmd) formatKanikoOutput(stdout io.ReadCloser, stderr io.ReadCloser) string {
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

	kanikoLogRegex := regexp.MustCompile(`^time="(.*)" level=(.*) msg="(.*)"`)
	buildPrefix := "build >"

	printFormattedOutput := func(originalLine string) {
		line := []byte(originalLine)

		for _, outputFormat := range outputFormats {
			line = outputFormat.Regex.ReplaceAll(line, []byte(outputFormat.Replacement))
		}

		lineString := string(line)

		if len(line) != len(originalLine) {
			log.Done(buildPrefix + lineString)
		} else if kanikoLogRegex.Match(line) == false {
			log.Info(buildPrefix + ">> " + lineString)
		}

		lastLine = string(kanikoLogRegex.ReplaceAll([]byte(originalLine), []byte("$3")))
	}

	processutil.RunOnEveryLine(stdout, printFormattedOutput, 500, wg)
	processutil.RunOnEveryLine(stderr, printFormattedOutput, 500, wg)

	wg.Wait()

	return lastLine
}

func (cmd *UpCmd) initRegistry() {
	log.StartWait("Initializing helm client")
	err := cmd.initHelm()
	log.StopWait()

	if err != nil {
		log.Fatalf("Error initializing helm client: %s", err.Error())
	}

	log.Done("Initialized helm client")

	installRegistry := cmd.flags.initRegistry

	if installRegistry {
		log.StartWait("Initializing docker registry")
		defer log.StopWait()

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
		_, err := cmd.helm.InstallChartByName(registryReleaseName, registryReleaseNamespace, "stable/docker-registry", "", &registryConfig)

		if err != nil {
			log.Panicf("Unable to initialize docker registry: %s", err.Error())
		}

		htpasswdSecretName := registryReleaseName + "-docker-registry-secret"
		htpasswdSecret, err := cmd.kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		if err != nil {
			log.Panicf("Unable to retrieve secret for docker registry: %s", err.Error())
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
		err = newHtpasswdData.SetPassword(registryUser.Username, registryUser.Password, htpasswd.HashBCrypt)

		if err != nil {
			log.Panicf("Unable to set password in htpasswd: %s", err.Error())
		}

		newHtpasswdDataBytes := newHtpasswdData.Bytes()

		htpasswdSecret.Data["htpasswd"] = newHtpasswdDataBytes

		_, err = cmd.kubectl.Core().Secrets(registryReleaseNamespace).Get(htpasswdSecretName, metav1.GetOptions{})

		if err != nil {
			_, err = cmd.kubectl.Core().Secrets(registryReleaseNamespace).Create(htpasswdSecret)
		} else {
			_, err = cmd.kubectl.Core().Secrets(registryReleaseNamespace).Update(htpasswdSecret)
		}

		if err != nil {
			log.Panicf("Unable to update htpasswd secret: %s", err.Error())
		}

		registryAuthEncoded := base64.StdEncoding.EncodeToString([]byte(cmd.privateConfig.Registry.User.Username + ":" + cmd.privateConfig.Registry.User.Password))
		registryServiceName := registryReleaseName + "-docker-registry"

		var registryService *k8sv1.Service

		maxServiceWaiting := 60 * time.Second
		serviceWaitingInterval := 3 * time.Second

		log.StopWait()
		log.Done("Initialized docker registry")

		log.StartWait("Waiting for docker registry to start")

		for true {
			registryService, err = cmd.kubectl.Core().Services(registryReleaseNamespace).Get(registryServiceName, metav1.GetOptions{})

			if err != nil {
				log.Panic(err)
			}

			if len(registryService.Spec.ClusterIP) > 0 {
				break
			}

			time.Sleep(serviceWaitingInterval)
			maxServiceWaiting = maxServiceWaiting - serviceWaitingInterval

			if maxServiceWaiting <= 0 {
				log.Panic("Timeout waiting for registry service to start")
			}
		}

		log.StopWait()
		log.Done("Docker registry started")

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

		_, err = cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Get(pullSecretName, metav1.GetOptions{})

		if err != nil {
			_, err = cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Create(registryPullSecret)
		} else {
			_, err = cmd.kubectl.Core().Secrets(cmd.privateConfig.Release.Namespace).Update(registryPullSecret)
		}

		if err != nil {
			log.Panicf("Unable to update image pull secret: %s", err.Error())
		}
	}
}

func (cmd *UpCmd) initHelm() error {
	if cmd.helm == nil {
		client, err := helmClient.NewClient(cmd.kubectl, false)

		if err != nil {
			return err
		}

		cmd.helm = client
	}

	return nil
}

func (cmd *UpCmd) deployChart() {
	if cmd.helm == nil {
		log.StartWait("Initializing helm client")
		err := cmd.initHelm()
		log.StopWait()

		if err != nil {
			log.Panic(err)
		}

		log.Done("Initialized helm client")
	}

	log.StartWait("Deploying helm chart")

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

	appRelease, err := cmd.helm.InstallChartByPath(releaseName, releaseNamespace, chartPath, &values)

	log.StopWait()

	if err != nil {
		log.Panicf("Unable to deploy helm chart: %s", err.Error())
	}

	releaseRevision := int(appRelease.Version)
	log.Donef("Deployed helm chart (Release revision: %d)", releaseRevision)
	log.StartWait("Waiting for release pod to become ready")
	defer log.StopWait()

	for true {
		podList, err := cmd.kubectl.Core().Pods(releaseNamespace).List(metav1.ListOptions{
			LabelSelector: "release=" + releaseName,
		})

		if err != nil {
			log.Panicf("Unable to list devspace pods: %s", err.Error())
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
				err = waitForPodReady(cmd.kubectl, cmd.pod, 2*60*time.Second, 5*time.Second)

				if err != nil {
					log.Panicf("Error during waiting for pod: %s", err.Error())
				}

				break
			} else {
				log.Info("Waiting for release upgrade to complete.")
			}
		} else {
			log.Info("Waiting for release to be deployed.")
		}
		time.Sleep(2 * time.Second)
	}

	log.StopWait()
}

func (cmd *UpCmd) startSync() {
	for _, syncPath := range cmd.dsConfig.SyncPaths {
		absLocalPath, err := filepath.Abs(cmd.workdir + syncPath.LocalSubPath)

		if err != nil {
			log.Panicf("Unable to resolve localSubPath %s: %s", syncPath.LocalSubPath, err.Error())
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
					log.Errorf("Unable to list devspace pods: %s", podListErr.Error())
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

	_, _, _, terminalErr := kubectl.Exec(cmd.kubectl, cmd.pod, cmd.pod.Spec.Containers[0].Name, shell, true, nil)

	if terminalErr != nil {
		if _, ok := terminalErr.(exec.CodeExitError); ok == false {
			log.Panicf("Unable to start terminal session: %s", terminalErr.Error())
		}
	}
}

func waitForPodReady(kubectl *kubernetes.Clientset, pod *k8sv1.Pod, maxWaitTime time.Duration, checkInterval time.Duration) error {
	for maxWaitTime > 0 {
		pod, err := kubectl.Core().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		}

		time.Sleep(checkInterval)
		maxWaitTime = maxWaitTime - checkInterval
	}

	return fmt.Errorf("Max wait time expired")
}
