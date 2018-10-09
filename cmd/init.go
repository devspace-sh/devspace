package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/util/kubeconfig"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/devspace/cloud"

	"github.com/covexo/devspace/pkg/devspace/builder/docker"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/spf13/cobra"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags           *InitCmdFlags
	workdir         string
	chartGenerator  *generator.ChartGenerator
	config          *v1.Config
	overwriteConfig *v1.Config
	defaultImage    *v1.ImageConfig
	defaultRegistry *v1.RegistryConfig
}

// InitCmdFlags are the flags available for the init-command
type InitCmdFlags struct {
	reconfigure      bool
	overwrite        bool
	templateRepoURL  string
	templateRepoPath string
	language         string
}

// InitCmdFlagsDefault are the default flags for InitCmdFlags
var InitCmdFlagsDefault = &InitCmdFlags{
	reconfigure:      false,
	overwrite:        false,
	templateRepoURL:  "https://github.com/covexo/devspace-templates.git",
	templateRepoPath: "",
	language:         "",
}

func init() {
	cmd := &InitCmd{
		flags: InitCmdFlagsDefault,
	}
	cobraCmd := &cobra.Command{
		Use:   "init",
		Short: "Initializes your DevSpace",
		Long: `
#######################################################
#################### devspace init ####################
#######################################################
Gets your project ready to start a DevSpaces.
Creates the following files and directories:

YOUR_PROJECT_PATH/
|
|-- Dockerfile
|
|-- chart/
|   |-- Chart.yaml
|   |-- values.yaml
|   |-- templates/
|       |-- deployment.yaml
|       |-- service.yaml
|       |-- ingress.yaml
|
|-- .devspace/
|   |-- .gitignore
|   |-- cluster.yaml
|   |-- config.yaml

#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVarP(&cmd.flags.reconfigure, "reconfigure", "r", cmd.flags.reconfigure, "Change existing configuration")
	cobraCmd.Flags().BoolVarP(&cmd.flags.overwrite, "overwrite", "o", cmd.flags.overwrite, "Overwrite existing chart files and Dockerfile")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoURL, "templateRepoUrl", cmd.flags.templateRepoURL, "Git repository for chart templates")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoPath, "templateRepoPath", cmd.flags.templateRepoPath, "Local path for cloning chart template repository (uses temp folder if not specified)")
	cobraCmd.Flags().StringVarP(&cmd.flags.language, "language", "l", cmd.flags.language, "Programming language of your project")
}

// Run executes the command logic
func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	log.StartFileLogging()

	workdir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Unable to determine current workdir: %s", err.Error())
	}

	cmd.workdir = workdir
	configExists, _ := configutil.ConfigExists()

	if configExists {
		cmd.config = configutil.GetConfig(false)
	} else {
		cmd.config = configutil.GetConfigInstance()
	}

	configutil.Merge(cmd.config, &v1.Config{
		Version: configutil.String("v1"),
		DevSpace: &v1.DevSpaceConfig{
			Release: &v1.Release{
				Name:      configutil.String("devspace"),
				Namespace: configutil.String("default"),
			},
		},
		Images: &map[string]*v1.ImageConfig{
			"default": &v1.ImageConfig{
				Name: configutil.String("devspace"),
				Build: &v1.BuildConfig{
					Engine: &v1.BuildEngine{
						Docker: &v1.DockerBuildEngine{
							Enabled: configutil.Bool(true),
						},
					},
				},
				Registry: configutil.String("default"),
			},
		},
		Registries: &map[string]*v1.RegistryConfig{
			"default": &v1.RegistryConfig{
				Auth: &v1.RegistryAuth{},
			},
			"internal": &v1.RegistryConfig{
				Auth: &v1.RegistryAuth{},
			},
		},
	})
	cmd.overwriteConfig = configutil.GetOverwriteConfig(false)

	imageMap := *cmd.config.Images
	cmd.defaultImage, _ = imageMap["default"]

	registryMap := *cmd.config.Registries
	cmd.defaultRegistry, _ = registryMap["default"]

	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite

	if !cmd.flags.overwrite {
		_, chartDirNotFound := os.Stat(cmd.workdir + "/chart")

		if chartDirNotFound == nil {
			overwriteAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to overwrite the existing files in /chart? (yes | no)",
				DefaultValue:           "no",
				ValidationRegexPattern: "^(yes)|(no)$",
			})
			createChart = (*overwriteAnswer == "yes")
		} else {
			createChart = true
		}
	}

	if createChart {
		cmd.initChartGenerator()
		cmd.determineLanguage()
		cmd.createChart()
	}

	if cmd.flags.reconfigure || !configExists {
		cmd.configureKubernetes()
		cmd.configureDevSpace()

		cmd.defaultImage.Name = cmd.config.DevSpace.Release.Name

		cmd.configureTiller()
		cmd.configureRegistry()

		err := configutil.SaveConfig()

		if err != nil {
			log.With(err).Fatalf("Config error: %s", err.Error())
		}

		_ = configutil.GetConfig(true)
	}
}

func (cmd *InitCmd) initChartGenerator() {
	templateRepoPath := cmd.flags.templateRepoPath

	if len(templateRepoPath) == 0 {
		templateRepoPath, _ = ioutil.TempDir("", "")
		defer os.RemoveAll(templateRepoPath)
	}
	templateRepo := &generator.TemplateRepository{
		URL:       cmd.flags.templateRepoURL,
		LocalPath: templateRepoPath,
	}
	cmd.chartGenerator = &generator.ChartGenerator{
		TemplateRepo: templateRepo,
		Path:         cmd.workdir,
	}
}

func (cmd *InitCmd) configureDevSpace() {
	_, chartDirNotFound := os.Stat(cmd.workdir + "/chart")
	if chartDirNotFound == nil {
		/*TODO
		existingChartYaml := map[interface{}]interface{}{}
		existingChartValuesYaml := map[interface{}]interface{}{}

		yamlutil.ReadYamlFromFile(cmd.workdir+"/chart/Chart.yaml", existingChartYaml)
		yamlutil.ReadYamlFromFile(cmd.workdir+"/chart/values.yaml", existingChartValuesYaml)

		cmd.config.Release.Name = existingChartYaml["name"].(string)

		applicationValues, applicationValuesCorrect := existingChartValuesYaml["container"].(map[interface{}]interface{})
		externalValues, externalValuesCorrect := existingChartValuesYaml["external"].(map[interface{}]interface{})

		if externalValuesCorrect {
			value, isCorrect := externalValues["domain"].(string)

			if isCorrect {
				cmd.appConfig.External.Domain = value
			}
		}*/
	}

	cmd.config.DevSpace.Release.Name = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "What is the name of your application?",
		DefaultValue:           *cmd.config.DevSpace.Release.Name,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})

	ports := strings.Split(*stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which port(s) does your application listen on? (separated by spaces)",
		DefaultValue:           "",
		ValidationRegexPattern: "^([1-9][0-9]{0,4})?(\\s[1-9][0-9]{0,4})*?$",
	}), " ")

	for _, port := range ports {
		portInt, _ := strconv.Atoi(port)

		if portInt > 0 {
			cmd.addPortForwarding(portInt)
		}
	}
	cmd.addDefaultSyncConfig()

	if cmd.config.DevSpace.Release.Namespace == nil || len(*cmd.config.DevSpace.Release.Namespace) == 0 {
		cmd.config.DevSpace.Release.Namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which Kubernetes namespace should your application run in?",
			DefaultValue:           *cmd.config.DevSpace.Release.Namespace,
			ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
		})
	}

	/* TODO
	cmd.appConfig.External.Domain = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which domain do you want to run your application on?",
		DefaultValue:           cmd.appConfig.External.Domain,
		ValidationRegexPattern: "^([a-z0-9]([a-z0-9-]{0,120}[a-z0-9])?\\.)+[a-z0-9]{2,}$",
	})*/
}

func (cmd *InitCmd) addPortForwarding(port int) {
	for _, portForwarding := range *cmd.config.DevSpace.PortForwarding {
		for _, portMapping := range *portForwarding.PortMappings {
			if *portMapping.RemotePort == port {
				return
			}
		}
	}

	portForwarding := append(*cmd.config.DevSpace.PortForwarding, &v1.PortForwardingConfig{
		PortMappings: &[]*v1.PortMapping{
			{
				LocalPort:  &port,
				RemotePort: &port,
			},
		},
		ResourceType: configutil.String("pod"),
		LabelSelector: &map[string]*string{
			"release": cmd.config.DevSpace.Release.Name,
		},
	})
	cmd.config.DevSpace.PortForwarding = &portForwarding
}

func (cmd *InitCmd) addDefaultSyncConfig() {
	for _, syncPath := range *cmd.config.DevSpace.Sync {
		if *syncPath.LocalSubPath == "./" || *syncPath.ContainerPath == "/app" {
			return
		}
	}
	dockerignoreFile := filepath.Join(cmd.workdir, ".dockerignore")
	dockerignore, err := ioutil.ReadFile(dockerignoreFile)
	uploadExcludePaths := []string{}

	if err == nil {
		dockerignoreRules := strings.Split(string(dockerignore), "\n")

		for _, ignoreRule := range dockerignoreRules {
			if len(ignoreRule) > 0 {
				uploadExcludePaths = append(uploadExcludePaths, ignoreRule)
			}
		}
	}

	syncConfig := append(*cmd.config.DevSpace.Sync, &v1.SyncConfig{
		ContainerPath: configutil.String("/app"),
		LocalSubPath:  configutil.String("./"),
		ResourceType:  configutil.String("pod"),
		LabelSelector: &map[string]*string{
			"release": cmd.config.DevSpace.Release.Name,
		},
		UploadExcludePaths: &uploadExcludePaths,
	})
	cmd.config.DevSpace.Sync = &syncConfig
}

func (cmd *InitCmd) configureTiller() {
	tillerConfig := cmd.config.Services.Tiller
	tillerRelease := tillerConfig.Release

	if tillerRelease.Namespace == nil || len(*tillerRelease.Namespace) == 0 {
		tillerRelease.Namespace = cmd.config.DevSpace.Release.Namespace

		tillerRelease.Namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which Kubernetes namespace should your tiller server run in?",
			DefaultValue:           *tillerRelease.Namespace,
			ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
		})
	}
}

func (cmd *InitCmd) useCloudProvider() bool {
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		log.Fatalf("Error loading cloud config: %v", err)
	}

	if len(providerConfig) > 1 {
		cloudProvider := "("

		for name := range providerConfig {
			if len(cloudProvider) > 1 {
				cloudProvider += ", "
			}

			cloudProvider += name
		}

		cloudProvider += ")"
		cloudProviderSelected := ""

		for ok := false; ok == false && cloudProviderSelected != "no"; {
			cloudProviderSelected = strings.TrimSpace(*stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:     "Do you want to use a cloud provider? (no to skip) " + cloudProvider,
				DefaultValue: cloud.DevSpaceCloudProviderName,
			}))

			_, ok = providerConfig[cloudProviderSelected]
		}

		if cloudProviderSelected != "no" {
			addToContext := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to add the cloud provider to the $HOME/.kube/config file? (yes | no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			}) == "yes"

			cmd.config.Cluster.CloudProvider = &cloudProviderSelected
			cmd.config.Cluster.UseKubeConfig = &addToContext

			log.StartWait("Logging into cloud provider " + providerConfig[cloudProviderSelected].Host + cloud.LoginEndpoint + "...")
			err := cloud.Update(providerConfig, cmd.config, true)
			log.StopWait()
			if err != nil {
				log.Fatalf("Couldn't authenticate to devspace cloud: %v", err)
			}

			return true
		}
	} else {
		useDevSpaceCloud := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to use the devspace cloud? (free ready-to-use kubernetes) (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		}) == "yes"

		if useDevSpaceCloud {
			addToContext := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to add the devspace-cloud to the $HOME/.kube/config file? (yes | no)",
				DefaultValue:           "yes",
				ValidationRegexPattern: "^(yes)|(no)$",
			}) == "yes"

			cmd.config.Cluster.CloudProvider = configutil.String(cloud.DevSpaceCloudProviderName)
			cmd.config.Cluster.UseKubeConfig = &addToContext

			log.StartWait("Logging into cloud provider " + providerConfig[cloud.DevSpaceCloudProviderName].Host + cloud.LoginEndpoint + "...")
			err := cloud.Update(providerConfig, cmd.config, true)
			log.StopWait()
			if err != nil {
				log.Fatalf("Couldn't authenticate to devspace cloud: %v", err)
			}

			return true
		}
	}

	return false
}

func (cmd *InitCmd) configureKubernetes() {
	clusterConfig := cmd.config.Cluster
	useKubeConfig := false

	// Check if devspace cloud should be used
	if cmd.useCloudProvider() {
		return
	}

	_, err := os.Stat(clientcmd.RecommendedHomeFile)
	if err == nil {
		skipAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to use your existing $HOME/.kube/config for Kubernetes access? (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		})

		useKubeConfig = (*skipAnswer == "yes")
	}

	clusterConfig.UseKubeConfig = configutil.Bool(useKubeConfig)
	if !useKubeConfig {
		if clusterConfig.APIServer == nil {
			clusterConfig.APIServer = configutil.String("https://192.168.99.100:8443")
		}
		clusterConfig.APIServer = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is your Kubernetes API Server URL? (e.g. https://127.0.0.1:8443)",
			DefaultValue:           *clusterConfig.APIServer,
			ValidationRegexPattern: "^https?://[a-z0-9-.]{0,99}:[0-9]{1,5}$",
		})

		if clusterConfig.CaCert == nil {
			clusterConfig.CaCert = configutil.String("")
		}
		clusterConfig.CaCert = stdinutil.AskChangeQuestion(&stdinutil.GetFromStdinParams{
			Question:               "What is the CA Certificate of your API Server? (PEM)",
			DefaultValue:           *clusterConfig.CaCert,
			InputTerminationString: "-----END CERTIFICATE-----",
		})

		if clusterConfig.User == nil {
			clusterConfig.User = &v1.ClusterUser{
				ClientCert: configutil.String(""),
				ClientKey:  configutil.String(""),
			}
		} else {
			if clusterConfig.User.ClientCert == nil {
				clusterConfig.User.ClientCert = configutil.String("")
			}

			if clusterConfig.User.ClientKey == nil {
				clusterConfig.User.ClientKey = configutil.String("")
			}
		}
		clusterConfig.User.ClientCert = stdinutil.AskChangeQuestion(&stdinutil.GetFromStdinParams{
			Question:               "What is your Kubernetes client certificate? (PEM)",
			DefaultValue:           *clusterConfig.User.ClientCert,
			InputTerminationString: "-----END CERTIFICATE-----",
		})
		clusterConfig.User.ClientKey = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is your Kubernetes client key? (RSA, PEM)",
			DefaultValue:           *clusterConfig.User.ClientKey,
			InputTerminationString: "-----END RSA PRIVATE KEY-----",
		})
	} else {
		currentContext, err := kubeconfig.GetCurrentContext()
		if err != nil {
			log.Fatalf("Couldn't determine current kubernetes context: %v", err)
		}

		clusterConfig.KubeContext = &currentContext
	}
}

func (cmd *InitCmd) configureRegistry() {
	dockerUsername := ""
	createInternalRegistryDefaultAnswer := "yes"

	var imageBuilder *docker.Builder
	var dockerBuilderErr error

	imageBuilder, dockerBuilderErr = docker.NewBuilder("", "", "", false)

	if dockerBuilderErr == nil {
		log.StartWait("Checking Docker credentials")
		dockerAuthConfig, dockerAuthErr := imageBuilder.Authenticate("", "", true)
		log.StopWait()

		if dockerAuthErr == nil {
			dockerUsername = dockerAuthConfig.Username

			if dockerUsername != "" {
				createInternalRegistryDefaultAnswer = "no"
			}
		}
	}
	internalRegistryConfig := cmd.config.Services.InternalRegistry
	createInternalRegistry := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Should we create a private registry within your Kubernetes cluster for you? (yes | no)",
		DefaultValue:           createInternalRegistryDefaultAnswer,
		ValidationRegexPattern: "^(yes)|(no)$",
	})

	if *createInternalRegistry == "no" {
		registryURL := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which registry do you want to push to? ('hub.docker.com' or URL)",
			DefaultValue:           "hub.docker.com",
			ValidationRegexPattern: "^.*$",
		})

		cmd.defaultRegistry.URL = registryURL
		internalRegistryConfig = nil
		loginWarningServer := ""

		if dockerUsername == "" {
			if *registryURL != "hub.docker.com" {
				loginWarningServer = " " + *registryURL
				imageBuilder, dockerBuilderErr = docker.NewBuilder(*registryURL, "", "", false)
			}

			if dockerBuilderErr == nil {
				log.StartWait("Checking Docker credentials")
				dockerAuthConfig, dockerAuthErr := imageBuilder.Authenticate("", "", true)
				log.StopWait()

				if dockerAuthErr == nil {
					dockerUsername = dockerAuthConfig.Username
				}
			}
		}
		googleRegistryRegex := regexp.MustCompile("^(.+\\.)?gcr.io$")
		isGoogleRegistry := googleRegistryRegex.Match([]byte(*registryURL))
		isDockerHub := *registryURL == "hub.docker.com"

		if dockerUsername == "" {
			if cmd.defaultImage.Build.Engine.Docker != nil {
				log.Fatal("Make sure you login to the registry with: docker login" + loginWarningServer)
			} else {
				registryMapOverwrite := *cmd.overwriteConfig.Registries
				defaultRegistryOverwrite, defaultRegistryOverwriteDefined := registryMapOverwrite["default"]

				if !defaultRegistryOverwriteDefined {
					defaultRegistryOverwrite = &v1.RegistryConfig{}
					registryMapOverwrite["default"] = defaultRegistryOverwrite
				}

				if defaultRegistryOverwrite.Auth == nil {
					defaultRegistryOverwrite.Auth = &v1.RegistryAuth{}
				}

				if defaultRegistryOverwrite.Auth.Username == nil {
					defaultRegistryOverwrite.Auth.Username = configutil.String("")
				}

				defaultRegistryOverwrite.Auth.Username = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Which username do you want to use to push images to " + *registryURL + "?",
					DefaultValue:           *defaultRegistryOverwrite.Auth.Username,
					ValidationRegexPattern: "^[a-zA-Z0-9]{4,30}$",
				})
				dockerUsername = *defaultRegistryOverwrite.Auth.Username

				if defaultRegistryOverwrite.Auth.Password == nil {
					defaultRegistryOverwrite.Auth.Password = configutil.String("")
				}

				defaultRegistryOverwrite.Auth.Username = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "Which password do you want to use to push images to " + *registryURL + "?",
					DefaultValue:           *defaultRegistryOverwrite.Auth.Password,
					ValidationRegexPattern: "^.*$",
				})
			}
		}
		defaultImageName := *cmd.defaultImage.Name
		defaultImageNameParts := strings.Split(defaultImageName, "/")

		if isDockerHub {
			if len(defaultImageNameParts) < 2 {
				defaultImageName = dockerUsername + "/" + strings.TrimPrefix(defaultImageName, dockerUsername)
			}

			cmd.defaultImage.Name = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Which image name do you want to use on Docker Hub?",
				DefaultValue:           defaultImageName,
				ValidationRegexPattern: "^[a-zA-Z0-9/]{4,30}$",
			})
		}

		if isGoogleRegistry {
			if len(defaultImageNameParts) < 2 {
				project, err := exec.Command("gcloud", "config", "get-value", "project").Output()
				gcloudProject := ""

				if err == nil {
					gcloudProject = strings.TrimSpace(string(project))
				}

				gcloudProjectName := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
					Question:               "What Google Cloud Project should be used?",
					DefaultValue:           gcloudProject,
					ValidationRegexPattern: "^.*$",
				})

				cmd.defaultImage.Name = configutil.String(*gcloudProjectName + "/" + strings.TrimPrefix(defaultImageName, *gcloudProjectName))
			}
		}
	} else {
		imageMap := *cmd.config.Images
		defaultImageConf, defaultImageExists := imageMap["default"]

		if defaultImageExists {
			defaultImageConf.Registry = configutil.String("internal")
		}

		if internalRegistryConfig == nil {
			internalRegistryConfig = &v1.InternalRegistry{
				Release: &v1.Release{},
			}
			cmd.config.Services.InternalRegistry = internalRegistryConfig
		}

		if internalRegistryConfig.Release.Name == nil {
			internalRegistryConfig.Release.Name = configutil.String("devspace-registry")
		}

		if internalRegistryConfig.Release.Namespace == nil {
			internalRegistryConfig.Release.Namespace = cmd.config.DevSpace.Release.Namespace
		}
		overwriteRegistryMap := *cmd.overwriteConfig.Registries

		overwriteRegistryConfig, overwriteRegistryConfigFound := overwriteRegistryMap["internal"]

		if !overwriteRegistryConfigFound {
			overwriteRegistryConfig = &v1.RegistryConfig{
				Auth: &v1.RegistryAuth{},
			}
			overwriteRegistryMap["internal"] = overwriteRegistryConfig
		}
		registryAuth := overwriteRegistryConfig.Auth

		if registryAuth.Username == nil {
			randomUserSuffix, err := randutil.GenerateRandomString(5)

			if err != nil {
				log.Fatalf("Error creating random username: %s", err.Error())
			}
			registryAuth.Username = configutil.String("user-" + randomUserSuffix)
		}

		if registryAuth.Password == nil {
			randomPassword, err := randutil.GenerateRandomString(12)

			if err != nil {
				log.Fatalf("Error creating random password: %s", err.Error())
			}
			registryAuth.Password = &randomPassword
		}
		var registryReleaseValues map[interface{}]interface{}

		if internalRegistryConfig.Release.Values != nil {
			registryReleaseValues = *internalRegistryConfig.Release.Values
		} else {
			registryReleaseValues = map[interface{}]interface{}{}

			registryDomain := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Which domain should your container registry be using? (optional, requires an ingress controller)",
				ValidationRegexPattern: "^(([a-z0-9]([a-z0-9-]{0,120}[a-z0-9])?\\.)+[a-z0-9]{2,})?$",
			})

			if *registryDomain != "" {
				registryReleaseValues = map[interface{}]interface{}{
					"Ingress": map[string]interface{}{
						"Enabled": true,
						"Hosts": []string{
							*registryDomain,
						},
						"Annotations": map[string]string{
							"Kubernetes.io/tls-acme": "true",
						},
						"Tls": []map[string]interface{}{
							map[string]interface{}{
								"SecretName": "tls-devspace-registry",
								"Hosts": []string{
									*registryDomain,
								},
							},
						},
					},
				}
			} else if kubectl.IsMinikube() == false {
				log.Warn("Your Kubernetes cluster will not be able to pull images from a registry without a registry domain!\n")
			}
		}
		secrets, registryHasSecrets := registryReleaseValues["secrets"]

		if !registryHasSecrets {
			secrets = map[interface{}]interface{}{}
			registryReleaseValues["secrets"] = secrets
		}
		secretMap, secretsIsMap := secrets.(map[interface{}]interface{})

		if secretsIsMap {
			_, registryHasSecretHtpasswd := secretMap["htpasswd"]

			if !registryHasSecretHtpasswd {
				secretMap["htpasswd"] = ""
			}
		}
		internalRegistryConfig.Release.Values = &registryReleaseValues
	}
}

func (cmd *InitCmd) determineLanguage() {
	if len(cmd.flags.language) != 0 {
		if cmd.chartGenerator.IsSupportedLanguage(cmd.flags.language) {
			cmd.chartGenerator.Language = cmd.flags.language
		} else {
			log.Info("Language '" + cmd.flags.language + "' not supported yet. Please open an issue here: https://github.com/covexo/devspace/issues/new?title=Feature%20Request:%20Language%20%22" + cmd.flags.language + "%22")
		}
	}

	if len(cmd.chartGenerator.Language) == 0 {
		log.StartWait("Detecting programming language")

		detectedLang := ""
		supportedLanguages, err := cmd.chartGenerator.GetSupportedLanguages()
		if err == nil {
			detectedLang, _ = cmd.chartGenerator.GetLanguage()
		}

		if detectedLang == "" {
			detectedLang = "none"
		}
		if len(supportedLanguages) == 0 {
			supportedLanguages = []string{"none"}
		}

		log.StopWait()

		cmd.chartGenerator.Language = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is the major programming language of your project?\nSupported languages: " + strings.Join(supportedLanguages, ", "),
			DefaultValue:           detectedLang,
			ValidationRegexPattern: "^(" + strings.Join(supportedLanguages, ")|(") + ")$",
		})
	}
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()

	if err != nil {
		log.Fatalf("Error while creating Helm chart and Dockerfile: %s", err.Error())
	}

	/*TODO
	createdChartYaml := map[interface{}]interface{}{}
	createdChartValuesYaml := map[interface{}]interface{}{}

	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/Chart.yaml", &createdChartYaml)
	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/values.yaml", &createdChartValuesYaml)

	containerValues, chartHasContainerValues := createdChartValuesYaml["container"].(map[interface{}]interface{})

	if !chartHasContainerValues && containerValues != nil {
		containerValues["port"] = cmd.appConfig.Container.Ports

		createdChartValuesYaml["container"] = containerValues
	}

	externalValues, chartHasExternalValues := createdChartValuesYaml["external"].(map[interface{}]interface{})

	if !chartHasExternalValues && externalValues != nil {
		externalValues["domain"] = cmd.appConfig.External.Domain
		createdChartValuesYaml["external"] = externalValues
	}
	yamlutil.WriteYamlToFile(createdChartYaml, cmd.chartGenerator.Path+"/chart/Chart.yaml")
	yamlutil.WriteYamlToFile(createdChartValuesYaml, cmd.chartGenerator.Path+"/chart/values.yaml")
	*/
}
