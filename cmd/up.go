package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/kaniko"
	"github.com/covexo/devspace/pkg/devspace/registry"
	synctool "github.com/covexo/devspace/pkg/devspace/sync"

	"github.com/covexo/devspace/pkg/devspace/config"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/exec"
)

// UpCmd is a struct that defines a command call for "up"
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

// UpCmdFlags are the flags available for the up-command
type UpCmdFlags struct {
	tiller           bool
	open             string
	initRegistry     bool
	build            bool
	shell            string
	sync             bool
	deploy           bool
	portforwarding   bool
	noSleep          bool
	imageDestination string
}

//UpFlagsDefault are the default flags for UpCmdFlags
var UpFlagsDefault = &UpCmdFlags{
	tiller:         true,
	open:           "cmd",
	initRegistry:   true,
	build:          true,
	sync:           true,
	deploy:         false,
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
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVar(&cmd.flags.tiller, "tiller", cmd.flags.tiller, "Install/upgrade tiller")
	cobraCmd.Flags().StringVarP(&cmd.flags.open, "open", "o", cmd.flags.open, "Install/upgrade tiller")
	cobraCmd.Flags().BoolVar(&cmd.flags.initRegistry, "init-registry", cmd.flags.initRegistry, "Install or upgrade Docker registry")
	cobraCmd.Flags().BoolVarP(&cmd.flags.build, "build", "b", cmd.flags.build, "Build image if Dockerfile has been modified")
	cobraCmd.Flags().StringVarP(&cmd.flags.shell, "shell", "s", "", "Shell command (default: bash, fallback: sh)")
	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "deploy", "d", cmd.flags.deploy, "Deploy chart")
	cobraCmd.Flags().BoolVar(&cmd.flags.noSleep, "no-sleep", cmd.flags.noSleep, "Enable no-sleep")
	cobraCmd.Flags().StringVar(&cmd.flags.imageDestination, "image-destination", "", "Choose image destination")
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

	var shouldRebuild bool

	if cmd.flags.build {
		shouldRebuild = cmd.shouldRebuild(cobraCmd.Flags().Changed("build"))

		if shouldRebuild {
			cmd.buildImage()

			// Save new image ip to config
			cmd.privateConfig.Release.LatestImage = cmd.latestImageIP
			err = config.SaveConfig(cmd.privateConfig)

			if err != nil {
				log.Fatalf("Config saving error: %s", err.Error())
			}
		} else {
			cmd.latestImageIP = cmd.privateConfig.Release.LatestImage
		}
	}

	if cmd.flags.deploy || shouldRebuild {
		cmd.deployChart()
	} else {
		cmd.initHelm()

		// Check if we find a running release pod
		pod, err := getRunningDevSpacePod(cmd.helm, cmd.kubectl, cmd.privateConfig)

		if err != nil {
			log.Fatalf("Couldn't find running devspace pod: %s", err.Error())
		}

		cmd.pod = pod
	}

	if cmd.flags.portforwarding {
		cmd.startPortForwarding()
	}

	if cmd.flags.sync {
		syncConfigs := cmd.startSync()
		defer func() {
			for _, v := range syncConfigs {
				v.Stop()
			}
		}()
	}

	cmd.enterTerminal()
}

func (cmd *UpCmd) shouldRebuild(buildFlagChanged bool) bool {
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
		if buildFlagChanged == false {
			if len(cmd.privateConfig.Release.LatestBuild) != 0 {
				latestBuildTime, _ := time.Parse(time.RFC3339Nano, cmd.privateConfig.Release.LatestBuild)

				// only rebuild Docker image when Dockerfile has changed since latest build
				mustRebuild = (latestBuildTime.Equal(dockerfileModTime) == false)
			}
		}
	}

	cmd.privateConfig.Release.LatestBuild = dockerfileModTime.Format(time.RFC3339Nano)

	return mustRebuild
}

func (cmd *UpCmd) buildImage() {
	cmd.initHelm()

	if cmd.flags.initRegistry && cmd.flags.imageDestination == "" {
		log.StartWait("Initializing docker registry")
		latestImageHostname, latestImageIP, err := registry.InitDockerRegistry(cmd.kubectl, cmd.helm, cmd.privateConfig, cmd.dsConfig)
		log.StopWait()

		if err != nil {
			log.Fatalf("Docker registry error: %s", err.Error())
		}

		cmd.latestImageHostname = latestImageHostname
		cmd.latestImageIP = latestImageIP

		log.Done("Docker registry started")
	}

	imageDestination := cmd.latestImageHostname

	if cmd.flags.imageDestination != "" {
		imageDestination = cmd.flags.imageDestination
	}

	err := kaniko.BuildDockerfile(cmd.kubectl, cmd.privateConfig.Release.Namespace, imageDestination, registry.PullSecretName)

	if err != nil {
		log.Fatalf("Image building failed: %s", err.Error())
	}
}

func (cmd *UpCmd) initHelm() {
	if cmd.helm == nil {
		log.StartWait("Initializing helm client")
		defer log.StopWait()

		client, err := helmClient.NewClient(cmd.kubectl, false)

		if err != nil {
			log.Fatalf("Error initializing helm client: %s", err.Error())
		}

		cmd.helm = client
		log.Done("Initialized helm client")
	}
}

func (cmd *UpCmd) deployChart() {
	cmd.initHelm()
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
		log.Fatalf("Unable to deploy helm chart: %s", err.Error())
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
				podRevision, podHasRevision := pod.Annotations["revision"]
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
			_, hasRevision := selectedPod.Annotations["revision"]

			if !hasRevision || highestRevision == releaseRevision {
				if !hasRevision {
					log.Warn("Found pod without revision. Use annotation 'revision' for your pods to avoid this warning.")
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

func (cmd *UpCmd) startSync() []*synctool.SyncConfig {
	syncConfigs := make([]*synctool.SyncConfig, 0, len(cmd.dsConfig.SyncPaths))

	for _, syncPath := range cmd.dsConfig.SyncPaths {
		absLocalPath, err := filepath.Abs(cmd.workdir + syncPath.LocalSubPath)

		if err != nil {
			log.Panicf("Unable to resolve localSubPath %s: %s", syncPath.LocalSubPath, err.Error())
		} else {
			// Retrieve pod from label selector
			labels := make([]string, 0, len(syncPath.LabelSelector))

			for key, value := range syncPath.LabelSelector {
				labels = append(labels, key+"="+value)
			}

			pod, err := kubectl.GetFirstRunningPod(cmd.kubectl, strings.Join(labels, ", "), cmd.privateConfig.Registry.Release.Namespace)

			if err != nil {
				log.Panicf("Unable to list devspace pods: %s", err.Error())
			} else if pod != nil {
				syncConfig := &synctool.SyncConfig{
					Kubectl:   cmd.kubectl,
					Pod:       pod,
					Container: &pod.Spec.Containers[0],
					WatchPath: absLocalPath,
					DestPath:  syncPath.ContainerPath,
				}

				err = syncConfig.Start()
				if err != nil {
					log.Fatalf("Sync error: %s", err.Error())
				}

				log.Donef("Sync started on %s <-> %s", absLocalPath, syncPath.ContainerPath)
				syncConfigs = append(syncConfigs, syncConfig)
			}
		}
	}

	return syncConfigs
}

func (cmd *UpCmd) startPortForwarding() {
	for _, portForwarding := range cmd.dsConfig.PortForwarding {
		if portForwarding.ResourceType == "pod" {
			if len(portForwarding.LabelSelector) > 0 {
				labels := make([]string, 0, len(portForwarding.LabelSelector))

				for key, value := range portForwarding.LabelSelector {
					labels = append(labels, key+"="+value)
				}

				pod, err := kubectl.GetFirstRunningPod(cmd.kubectl, strings.Join(labels, ", "), cmd.privateConfig.Registry.Release.Namespace)

				if err != nil {
					log.Errorf("Unable to list devspace pods: %s", err.Error())
				} else if pod != nil {
					ports := make([]string, len(portForwarding.PortMappings))

					for index, value := range portForwarding.PortMappings {
						ports[index] = strconv.Itoa(value.LocalPort) + ":" + strconv.Itoa(value.RemotePort)
					}

					readyChan := make(chan struct{})

					go kubectl.ForwardPorts(cmd.kubectl, pod, ports, make(chan struct{}), readyChan)

					// Wait till forwarding is ready
					select {
					case <-readyChan:
						log.Donef("Port forwarding started on %s", strings.Join(ports, ", "))
					case <-time.After(5 * time.Second):
						log.Error("Timeout waiting for port forwarding to start")
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
			log.Fatalf("Unable to start terminal session: %s", terminalErr.Error())
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
