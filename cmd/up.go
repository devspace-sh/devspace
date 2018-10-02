package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/devspace/login"

	"github.com/covexo/devspace/pkg/util/hash"
	"github.com/covexo/devspace/pkg/util/stdinutil"

	"github.com/covexo/devspace/pkg/util/yamlutil"

	"github.com/covexo/devspace/pkg/devspace/builder"

	"github.com/docker/docker/api/types"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/covexo/devspace/pkg/util/randutil"

	"github.com/covexo/devspace/pkg/util/log"

	"github.com/covexo/devspace/pkg/devspace/builder/kaniko"
	"github.com/covexo/devspace/pkg/devspace/registry"
	synctool "github.com/covexo/devspace/pkg/devspace/sync"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	k8sv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/client-go/kubernetes"
)

// UpCmd is a struct that defines a command call for "up"
type UpCmd struct {
	flags     *UpCmdFlags
	helm      *helmClient.HelmClientWrapper
	kubectl   *kubernetes.Clientset
	workdir   string
	pod       *k8sv1.Pod
	container *k8sv1.Container
}

// UpCmdFlags are the flags available for the up-command
type UpCmdFlags struct {
	tiller         bool
	open           string
	initRegistries bool
	build          bool
	sync           bool
	deploy         bool
	portforwarding bool
	noSleep        bool
	verboseSync    bool
	container      string
}

//UpFlagsDefault are the default flags for UpCmdFlags
var UpFlagsDefault = &UpCmdFlags{
	tiller:         true,
	open:           "cmd",
	initRegistries: true,
	build:          false,
	sync:           true,
	deploy:         false,
	portforwarding: true,
	noSleep:        false,
	verboseSync:    false,
	container:      "",
}

const clusterRoleBindingName = "devspace-users"

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
	cobraCmd.Flags().BoolVar(&cmd.flags.initRegistries, "init-registries", cmd.flags.initRegistries, "Initialize registries (and install internal one)")
	cobraCmd.Flags().BoolVarP(&cmd.flags.build, "build", "b", cmd.flags.build, "Force image build")
	cobraCmd.Flags().StringVarP(&cmd.flags.container, "container", "c", cmd.flags.container, "Container name where to open the shell")
	cobraCmd.Flags().BoolVar(&cmd.flags.sync, "sync", cmd.flags.sync, "Enable code synchronization")
	cobraCmd.Flags().BoolVar(&cmd.flags.verboseSync, "verbose-sync", cmd.flags.verboseSync, "When enabled the sync will log every file change")
	cobraCmd.Flags().BoolVar(&cmd.flags.portforwarding, "portforwarding", cmd.flags.portforwarding, "Enable port forwarding")
	cobraCmd.Flags().BoolVarP(&cmd.flags.deploy, "deploy", "d", cmd.flags.deploy, "Force chart deployment")
	cobraCmd.Flags().BoolVar(&cmd.flags.noSleep, "no-sleep", cmd.flags.noSleep, "Enable no-sleep (Override the containers.default.command and containers.default.args values with empty strings)")
}

// Run executes the command logic
func (cmd *UpCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	workdir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
	}

	cmd.workdir = workdir

	configExists, _ := configutil.ConfigExists()
	if !configExists {
		initCmd := &InitCmd{
			flags: InitCmdFlagsDefault,
		}

		initCmd.Run(nil, []string{})
	}

	// Load config
	config := configutil.GetConfig(false)
	if config.Cluster.DevSpaceCloud != nil && *config.Cluster.DevSpaceCloud {
		err = login.Update(config)
		if err != nil {
			log.Warnf("Couldn't update devspace cloud cluster information: %v", err)
		}
	}

	cmd.kubectl, err = kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	err = cmd.ensureNamespace()
	if err != nil {
		log.Fatalf("Unable to create release namespace: %v", err)
	}

	err = cmd.ensureClusterRoleBinding()
	if err != nil {
		log.Fatalf("Unable to create ClusterRoleBinding: %v", err)
	}

	cmd.initHelm()

	if cmd.flags.initRegistries {
		cmd.initRegistries()
	}

	// Build image if necessary
	mustRedeploy := cmd.buildImages()

	// Check if we find a running release pod
	hash, err := hash.Directory("chart")
	if err != nil {
		log.Fatalf("Error hashing chart directory: %v", err)
	}

	pod, err := getRunningDevSpacePod(cmd.helm, cmd.kubectl)
	if err != nil || mustRedeploy || cmd.flags.deploy || config.DevSpace.ChartHash == nil || *config.DevSpace.ChartHash != hash {
		cmd.deployChart()

		config.DevSpace.ChartHash = &hash

		err = configutil.SaveConfig()
		if err != nil {
			log.Fatalf("Error saving config: %v", err)
		}
	} else {
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

	enterTerminal(cmd.kubectl, cmd.pod, cmd.flags.container, args)
}

func (cmd *UpCmd) ensureNamespace() error {
	config := configutil.GetConfig(false)
	releaseNamespace := *config.DevSpace.Release.Namespace

	// Check if registry namespace exists
	_, err := cmd.kubectl.CoreV1().Namespaces().Get(releaseNamespace, metav1.GetOptions{})
	if err != nil {
		// Create registry namespace
		_, err = cmd.kubectl.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: releaseNamespace,
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *UpCmd) ensureClusterRoleBinding() error {
	/*
		config := configutil.GetConfig(false)

		accessReview := &k8sauthorizationv1.SelfSubjectAccessReview{
			Spec: k8sauthorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &k8sauthorizationv1.ResourceAttributes{
					Namespace: *config.DevSpace.Release.Namespace,
					Verb:      "create",
					Group:     "rbac.authorization.k8s.io",
					Resource:  "roles",
				},
			},
		}

		resp, permErr := cmd.kubectl.Authorization().SelfSubjectAccessReviews().Create(accessReview)

		if permErr != nil {*/

	if kubectl.IsMinikube() {
		return nil
	}

	_, err := cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Get(clusterRoleBindingName, metav1.GetOptions{})

	if err != nil {
		clusterConfig, _ := kubectl.GetClientConfig()

		if clusterConfig.AuthProvider != nil && clusterConfig.AuthProvider.Name == "gcp" {
			createRoleBinding := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want the ClusterRoleBinding '" + clusterRoleBindingName + "' to be created automatically? (yes|no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			})

			if *createRoleBinding == "no" {
				log.Fatal("Please create ClusterRoleBinding '" + clusterRoleBindingName + "' manually")
			}
			username := configutil.String("")

			log.StartWait("Checking gcloud account")
			gcloudOutput, gcloudErr := exec.Command("gcloud", "config", "list", "account", "--format", "value(core.account)").Output()
			log.StopWait()

			if gcloudErr == nil {
				gcloudEmail := strings.TrimSuffix(strings.TrimSuffix(string(gcloudOutput), "\r\n"), "\n")

				if gcloudEmail != "" {
					username = &gcloudEmail
				}
			}

			username = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "What is the email address of your Google Cloud account?",
				DefaultValue:           *username,
				ValidationRegexPattern: ".+",
			})

			rolebinding := &k8sv1beta1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleBindingName,
				},
				Subjects: []k8sv1beta1.Subject{
					{
						Kind: "User",
						Name: *username,
					},
				},
				RoleRef: k8sv1beta1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
			}

			_, roleBindingErr := cmd.kubectl.RbacV1beta1().ClusterRoleBindings().Create(rolebinding)
			if roleBindingErr != nil {
				return roleBindingErr
			}
		} else {
			log.Warn("Unable to check permissions: If you run into errors, please create the ClusterRoleBinding '" + clusterRoleBindingName + "' as described here: https://devspace.covexo.com/docs/advanced/rbac.html")
		}
	}
	return nil
}

func (cmd *UpCmd) initRegistries() {
	config := configutil.GetConfig(false)
	registryMap := *config.Registries

	if config.Services.InternalRegistry != nil {
		registryConf, regConfExists := registryMap["internal"]
		if !regConfExists {
			log.Fatal("Registry config not found for internal registry")
		}

		log.StartWait("Initializing internal registry")
		err := registry.InitInternalRegistry(cmd.kubectl, cmd.helm, config.Services.InternalRegistry, registryConf)
		log.StopWait()

		if err != nil {
			log.Fatalf("Internal registry error: %v", err)
		}

		err = configutil.SaveConfig()
		if err != nil {
			log.Fatalf("Saving config error: %v", err)
		}

		log.Done("Internal registry started")
	}

	for registryName, registryConf := range registryMap {
		if registryConf.Auth != nil && registryConf.Auth.Password != nil {
			username := ""
			password := *registryConf.Auth.Password
			email := "noreply@devspace-cloud.com"
			registryURL := ""

			if registryConf.Auth.Username != nil {
				username = *registryConf.Auth.Username
			}
			if registryConf.URL != nil {
				registryURL = *registryConf.URL
			}

			log.StartWait("Creating image pull secret for registry: " + registryName)
			err := registry.CreatePullSecret(cmd.kubectl, *config.DevSpace.Release.Namespace, registryURL, username, password, email)
			log.StopWait()

			if err != nil {
				log.Fatalf("Failed to create pull secret for registry: %v", err)
			}
		}
	}
}

func (cmd *UpCmd) shouldRebuild(imageConf *v1.ImageConfig, dockerfilePath string) bool {
	var dockerfileModTime time.Time

	mustRebuild := true
	dockerfileInfo, err := os.Stat(dockerfilePath)

	if err != nil {
		if imageConf.Build.LatestTimestamp == nil {
			log.Fatalf("Dockerfile missing: %v", err)
		} else {
			mustRebuild = false
		}
	} else {
		dockerfileModTime = dockerfileInfo.ModTime()

		// When user has not used -b or --build flags
		if cmd.flags.build == false {
			if imageConf.Build.LatestTimestamp != nil {
				latestBuildTime, _ := time.Parse(time.RFC3339Nano, *imageConf.Build.LatestTimestamp)

				// only rebuild Docker image when Dockerfile has changed since latest build
				mustRebuild = (latestBuildTime.Equal(dockerfileModTime) == false)
			}
		}
	}

	imageConf.Build.LatestTimestamp = configutil.String(dockerfileModTime.Format(time.RFC3339Nano))
	return mustRebuild
}

// returns true when one of the images had to be rebuild
func (cmd *UpCmd) buildImages() bool {
	re := false
	config := configutil.GetConfig(false)

	for imageName, imageConf := range *config.Images {
		dockerfilePath := "./Dockerfile"
		contextPath := "./"

		if imageConf.Build.DockerfilePath != nil {
			dockerfilePath = *imageConf.Build.DockerfilePath
		}

		if imageConf.Build.ContextPath != nil {
			contextPath = *imageConf.Build.ContextPath
		}

		dockerfilePath, err := filepath.Abs(dockerfilePath)
		if err != nil {
			log.Fatalf("Couldn't determine absolute path for %s", *imageConf.Build.DockerfilePath)
		}

		contextPath, err = filepath.Abs(contextPath)
		if err != nil {
			log.Fatalf("Couldn't determine absolute path for %s", *imageConf.Build.ContextPath)
		}

		if cmd.shouldRebuild(imageConf, dockerfilePath) {
			re = true
			imageTag, randErr := randutil.GenerateRandomString(7)

			if randErr != nil {
				log.Fatalf("Image building failed: %s", randErr.Error())
			}
			registryConf, err := registry.GetRegistryConfig(imageConf)
			if err != nil {
				log.Fatal(err)
			}

			var imageBuilder builder.Interface

			buildInfo := "Building image '%s' with engine '%s'"
			engineName := ""
			registryURL := ""

			if registryConf.URL != nil {
				registryURL = *registryConf.URL
			}
			if registryURL == "hub.docker.com" {
				registryURL = ""
			}

			if imageConf.Build.Engine.Kaniko != nil {
				engineName = "kaniko"
				buildNamespace := *config.DevSpace.Release.Namespace
				allowInsecurePush := false

				if imageConf.Build.Engine.Kaniko.Namespace != nil {
					buildNamespace = *imageConf.Build.Engine.Kaniko.Namespace
				}

				if registryConf.Insecure != nil {
					allowInsecurePush = *registryConf.Insecure
				}
				imageBuilder, err = kaniko.NewBuilder(registryURL, *imageConf.Name, imageTag, buildNamespace, cmd.kubectl, allowInsecurePush)
				if err != nil {
					log.Fatalf("Error creating kaniko builder: %v", err)
				}
			} else {
				engineName = "docker"
				preferMinikube := true

				if imageConf.Build.Engine.Docker.PreferMinikube != nil {
					preferMinikube = *imageConf.Build.Engine.Docker.PreferMinikube
				}

				imageBuilder, err = docker.NewBuilder(registryURL, *imageConf.Name, imageTag, preferMinikube)
				if err != nil {
					log.Fatalf("Error creating docker client: %v", err)
				}
			}

			log.Infof(buildInfo, imageName, engineName)

			if registryConf.URL != nil {
				registryURL = *registryConf.URL
			}

			username := ""
			password := ""
			if registryConf.Auth != nil {
				if registryConf.Auth.Username != nil {
					username = *registryConf.Auth.Username
				}

				if registryConf.Auth.Password != nil {
					password = *registryConf.Auth.Password
				}
			}

			log.StartWait("Authenticating (" + registryURL + ")")
			_, err = imageBuilder.Authenticate(username, password, len(username) == 0)
			log.StopWait()

			if err != nil {
				log.Fatalf("Error during image registry authentication: %v", err)
			}

			log.Done("Authentication successful (" + registryURL + ")")

			buildOptions := &types.ImageBuildOptions{}
			if imageConf.Build.Options != nil {
				if imageConf.Build.Options.BuildArgs != nil {
					buildOptions.BuildArgs = *imageConf.Build.Options.BuildArgs
				}
				if imageConf.Build.Options.Target != nil {
					buildOptions.Target = *imageConf.Build.Options.Target
				}
				if imageConf.Build.Options.Network != nil {
					buildOptions.NetworkMode = *imageConf.Build.Options.Network
				}
			}

			err = imageBuilder.BuildImage(contextPath, dockerfilePath, buildOptions)
			if err != nil {
				log.Fatalf("Error during image build: %v", err)
			}

			err = imageBuilder.PushImage()
			if err != nil {
				log.Fatalf("Error during image push: %v", err)
			}

			log.Info("Image pushed to registry (" + registryURL + ")")
			imageConf.Tag = &imageTag

			err = configutil.SaveConfig()
			if err != nil {
				log.Fatalf("Config saving error: %s", err.Error())
			}

			log.Done("Done building and pushing image '" + imageName + "'")
		} else {
			log.Infof("Skip building image '%s'", imageName)
		}
	}
	return re
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
	config := configutil.GetConfig(false)

	log.StartWait("Deploying helm chart")

	releaseName := *config.DevSpace.Release.Name
	releaseNamespace := *config.DevSpace.Release.Namespace
	chartPath := "chart/"

	values := map[interface{}]interface{}{}
	overwriteValues := map[interface{}]interface{}{}

	err := yamlutil.ReadYamlFromFile(chartPath+"values.yaml", values)
	if err != nil {
		log.Fatalf("Couldn't deploy chart, error reading from chart values %s: %v", chartPath+"values.yaml", err)
	}

	containerValues := map[string]interface{}{}

	for imageName, imageConf := range *config.Images {
		container := map[string]interface{}{}
		container["image"] = registry.GetImageURL(imageConf, true)

		if cmd.flags.noSleep {
			container["command"] = []string{}
			container["args"] = []string{}
		}

		containerValues[imageName] = container
	}

	pullSecrets := []interface{}{}
	existingPullSecrets, pullSecretsExisting := values["pullSecrets"]

	if pullSecretsExisting {
		pullSecrets = existingPullSecrets.([]interface{})
	}

	for _, registryConf := range *config.Registries {
		if registryConf.URL != nil {
			registrySecretName := registry.GetRegistryAuthSecretName(*registryConf.URL)
			pullSecrets = append(pullSecrets, registrySecretName)
		}
	}

	overwriteValues["containers"] = containerValues
	overwriteValues["pullSecrets"] = pullSecrets

	appRelease, err := cmd.helm.InstallChartByPath(releaseName, releaseNamespace, chartPath, &overwriteValues)

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
			var selectedPod *k8sv1.Pod

			for i, pod := range podList.Items {
				if kubectl.GetPodStatus(&pod) == "Terminating" {
					continue
				}

				podRevision, podHasRevision := pod.Annotations["revision"]
				hasHigherRevision := (i == 0)

				if !hasHigherRevision && podHasRevision {
					podRevisionInt, _ := strconv.Atoi(podRevision)

					if podRevisionInt > highestRevision {
						hasHigherRevision = true
					}
				}

				if hasHigherRevision {
					selectedPod = &pod
					highestRevision, _ = strconv.Atoi(podRevision)
				}
			}

			if selectedPod != nil {
				_, hasRevision := selectedPod.Annotations["revision"]

				if !hasRevision || highestRevision == releaseRevision {
					if !hasRevision {
						log.Warn("Found pod without revision. Use annotation 'revision' for your pods to avoid this warning.")
					}

					cmd.pod = selectedPod
					err = waitForPodReady(cmd.kubectl, cmd.pod, 2*60*time.Second, 5*time.Second)

					if err != nil {
						log.Fatalf("Error during waiting for pod: %s", err.Error())
					}

					break
				} else {
					log.Info("Waiting for release upgrade to complete.")
				}
			}
		} else {
			log.Info("Waiting for release to be deployed.")
		}

		time.Sleep(2 * time.Second)
	}

	log.StopWait()
}

func (cmd *UpCmd) startSync() []*synctool.SyncConfig {
	config := configutil.GetConfig(false)
	syncConfigs := make([]*synctool.SyncConfig, 0, len(*config.DevSpace.Sync))

	for _, syncPath := range *config.DevSpace.Sync {
		absLocalPath, err := filepath.Abs(*syncPath.LocalSubPath)

		if err != nil {
			log.Panicf("Unable to resolve localSubPath %s: %s", *syncPath.LocalSubPath, err.Error())
		} else {
			// Retrieve pod from label selector
			labels := make([]string, 0, len(*syncPath.LabelSelector))

			for key, value := range *syncPath.LabelSelector {
				labels = append(labels, key+"="+*value)
			}

			namespace := *config.DevSpace.Release.Namespace
			if syncPath.Namespace != nil && *syncPath.Namespace != "" {
				namespace = *syncPath.Namespace
			}

			pod, err := kubectl.GetFirstRunningPod(cmd.kubectl, strings.Join(labels, ", "), namespace)

			if err != nil {
				log.Panicf("Unable to list devspace pods: %s", err.Error())
			} else if pod != nil {
				if len(pod.Spec.Containers) == 0 {
					log.Warnf("Cannot start sync on pod, because selected pod %s/%s has no containers", pod.Namespace, pod.Name)
					continue
				}

				container := &pod.Spec.Containers[0]
				if syncPath.ContainerName != nil && *syncPath.ContainerName != "" {
					found := false

					for _, c := range pod.Spec.Containers {
						if c.Name == *syncPath.ContainerName {
							container = &c
							found = true
							break
						}
					}

					if found == false {
						log.Warnf("Couldn't start sync, because container %s wasn't found in pod %s/%s", *syncPath.ContainerName, pod.Namespace, pod.Name)
						continue
					}
				}

				syncConfig := &synctool.SyncConfig{
					Kubectl:   cmd.kubectl,
					Pod:       pod,
					Container: container,
					WatchPath: absLocalPath,
					DestPath:  *syncPath.ContainerPath,
					Verbose:   cmd.flags.verboseSync,
				}

				if syncPath.ExcludePaths != nil {
					syncConfig.ExcludePaths = *syncPath.ExcludePaths
				}

				if syncPath.DownloadExcludePaths != nil {
					syncConfig.DownloadExcludePaths = *syncPath.DownloadExcludePaths
				}

				if syncPath.UploadExcludePaths != nil {
					syncConfig.UploadExcludePaths = *syncPath.UploadExcludePaths
				}

				err = syncConfig.Start()
				if err != nil {
					log.Fatalf("Sync error: %s", err.Error())
				}

				log.Donef("Sync started on %s <-> %s (Pod: %s/%s)", absLocalPath, *syncPath.ContainerPath, pod.Namespace, pod.Name)
				syncConfigs = append(syncConfigs, syncConfig)
			}
		}
	}

	return syncConfigs
}

func (cmd *UpCmd) startPortForwarding() {
	config := configutil.GetConfig(false)

	for _, portForwarding := range *config.DevSpace.PortForwarding {
		if *portForwarding.ResourceType == "pod" {
			if len(*portForwarding.LabelSelector) > 0 {
				labels := make([]string, 0, len(*portForwarding.LabelSelector))

				for key, value := range *portForwarding.LabelSelector {
					labels = append(labels, key+"="+*value)
				}

				namespace := *config.DevSpace.Release.Namespace
				if portForwarding.Namespace != nil && *portForwarding.Namespace != "" {
					namespace = *portForwarding.Namespace
				}

				pod, err := kubectl.GetFirstRunningPod(cmd.kubectl, strings.Join(labels, ", "), namespace)

				if err != nil {
					log.Errorf("Unable to list devspace pods: %s", err.Error())
				} else if pod != nil {
					ports := make([]string, len(*portForwarding.PortMappings))

					for index, value := range *portForwarding.PortMappings {
						ports[index] = strconv.Itoa(*value.LocalPort) + ":" + strconv.Itoa(*value.RemotePort)
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
