package cmd

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/imdario/mergo"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/spf13/cobra"
)

// InitCmd is a struct that defines a command call for "init"
type InitCmd struct {
	flags          *InitCmdFlags
	workdir        string
	chartGenerator *generator.ChartGenerator
	config         *v1.Config
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
	mergo.Merge(cmd.config, &v1.Config{
		Version: configutil.String("v1"),
		DevSpace: &v1.DevSpaceConfig{
			Release: &v1.Release{
				Name:      configutil.String("devspace"),
				Namespace: configutil.String("default"),
			},
		},
		Image: &v1.ImageConfig{
			Name: configutil.String("devspace"),
		},
		Cluster: &v1.Cluster{
			APIServer: configutil.String("https://192.168.99.100:8443"),
			User:      &v1.User{},
		},
	})
	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite

	if !cmd.flags.overwrite {
		_, dockerfileNotFound := os.Stat(cmd.chartGenerator.Path + "/Dockerfile")
		_, chartDirNotFound := os.Stat(cmd.chartGenerator.Path + "/chart")

		if dockerfileNotFound == nil || chartDirNotFound == nil {
			overwriteAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
				Question:               "Do you want to overwrite the Dockerfile and the existing files in /chart? (yes | no)",
				DefaultValue:           "no",
				ValidationRegexPattern: "^(yes)|(no)$",
			})
			createChart = (*overwriteAnswer == "yes")
		} else {
			createChart = true
		}
	}

	if createChart {
		cmd.determineAppConfig()

		cmd.config.Image.Name = cmd.config.DevSpace.Release.Name
	}

	if cmd.flags.reconfigure || !configExists {
		cmd.reconfigure()
	}

	if createChart {
		cmd.determineLanguage()
		cmd.createChart()
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

func (cmd *InitCmd) determineAppConfig() {
	_, chartDirNotFound := os.Stat(cmd.chartGenerator.Path + "/chart")

	if chartDirNotFound == nil {
		/*TODO
		existingChartYaml := map[interface{}]interface{}{}
		existingChartValuesYaml := map[interface{}]interface{}{}

		yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/Chart.yaml", existingChartYaml)
		yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/values.yaml", existingChartValuesYaml)

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

	// cmd.appConfig.Container.Ports, _ = strconv.Atoi(stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
	// 	Question:               "Which port(s) does your application listen on? (separated by spaces)",
	// 	DefaultValue:           strconv.Itoa(cmd.appConfig.Container.Port),
	// 	ValidationRegexPattern: "^[1-9][0-9]{0,4}?(\\s[1-9][0-9]{0,4})?$",
	// }))

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

	syncConfig := append(*cmd.config.DevSpace.Sync, &v1.SyncConfig{
		ContainerPath: configutil.String("/app"),
		LocalSubPath:  configutil.String("./"),
		ResourceType:  configutil.String("pod"),
		LabelSelector: &map[string]*string{
			"release": cmd.config.DevSpace.Release.Name,
		},
	})
	cmd.config.DevSpace.Sync = &syncConfig
}

func (cmd *InitCmd) reconfigure() {
	clusterConfig := cmd.config.Cluster
	tillerConfig := cmd.config.Services.Tiller
	tillerRelease := tillerConfig.Release

	cmd.config.DevSpace.Release.Namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which Kubernetes namespace should your application run in?",
		DefaultValue:           *cmd.config.DevSpace.Release.Namespace,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})

	if tillerRelease.Namespace == nil {
		tillerRelease.Namespace = cmd.config.DevSpace.Release.Namespace
	}
	tillerRelease.Namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Which Kubernetes namespace should your tiller server run in?",
		DefaultValue:           *tillerRelease.Namespace,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})
	useKubeConfig := false
	homeDir, homeErr := homedir.Dir()

	if homeErr != nil {
		log.With(homeErr).Fatalf("Unable to determine home dir")
	}
	kubeConfigPath := homeDir + "/.kube/config"

	_, kubeConfigNotFound := os.Stat(kubeConfigPath)

	if kubeConfigNotFound == nil {
		skipAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to use your existing $HOME/.kube/config for Kubernetes access? (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		})
		useKubeConfig = (*skipAnswer == "yes")
	}
	clusterConfig.UseKubeConfig = configutil.Bool(useKubeConfig)

	if !useKubeConfig {
		clusterConfig.APIServer = stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is your Kubernetes API Server URL? (e.g. https://127.0.0.1:8443)",
			DefaultValue:           *clusterConfig.APIServer,
			ValidationRegexPattern: "^https?://[a-z0-9-.]{0,99}:[0-9]{1,5}$",
		})
		clusterConfig.CaCert = stdinutil.AskChangeQuestion(&stdinutil.GetFromStdinParams{
			Question:               "What is the CA Certificate of your API Server? (PEM)",
			DefaultValue:           *clusterConfig.CaCert,
			InputTerminationString: "-----END CERTIFICATE-----",
		})
		clusterConfig.User.Username = stdinutil.AskChangeQuestion(&stdinutil.GetFromStdinParams{
			Question:               "What is your Kubernetes username?",
			DefaultValue:           *clusterConfig.User.Username,
			ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
		})
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
	}
	cmd.reconfigureRegistry()

	err := configutil.SaveConfig()

	if err != nil {
		log.With(err).Fatalf("Config error: %s", err.Error())
	}
}

func (cmd *InitCmd) reconfigureRegistry() {
	overwriteConfig := configutil.GetOverwriteConfig()
	registryConfig := cmd.config.Services.Registry

	enableAutomaticBuilds := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Do you want to enable automatic Docker image building?",
		DefaultValue:           "yes",
		ValidationRegexPattern: "^(yes)|(no)$",
	})

	if *enableAutomaticBuilds == "yes" {
		internalRegistryKey := "internal registry"
		defaultRegistryValue := internalRegistryKey

		if registryConfig.External != nil {
			defaultRegistryValue = *registryConfig.External
		}
		registryURL := stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Which registry do you want to push to? (URL or 'internal registry')",
			DefaultValue:           defaultRegistryValue,
			ValidationRegexPattern: "^.*$",
		})

		if *registryURL != internalRegistryKey {
			registryConfig.External = registryURL
			registryConfig.Internal = nil
		} else {
			registryConfig.External = nil

			if registryConfig.Internal.Release.Name == nil {
				registryConfig.Internal.Release.Name = configutil.String("devspace-registry")
			}

			if registryConfig.Internal.Release.Namespace == nil {
				registryConfig.Internal.Release.Namespace = cmd.config.DevSpace.Release.Namespace
			}
			registryUser := overwriteConfig.Services.Registry.User

			if registryUser.Username == nil {
				randomUserSuffix, err := randutil.GenerateRandomString(5)

				if err != nil {
					log.Fatalf("Error creating random username: %s", err.Error())
				}
				registryUser.Username = configutil.String("user-" + randomUserSuffix)
			}

			if registryUser.Password == nil {
				randomPassword, err := randutil.GenerateRandomString(12)

				if err != nil {
					log.Fatalf("Error creating random password: %s", err.Error())
				}
				registryUser.Password = &randomPassword
			}
			var registryReleaseValues map[interface{}]interface{}

			if registryConfig.Internal.Release.Values != nil {
				registryReleaseValues = *registryConfig.Internal.Release.Values
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
			registryConfig.Internal.Release.Values = &registryReleaseValues
		}
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

		cmd.chartGenerator.Language, _ = cmd.chartGenerator.GetLanguage()
		supportedLanguages, err := cmd.chartGenerator.GetSupportedLanguages()

		if cmd.chartGenerator.Language == "" {
			cmd.chartGenerator.Language = "none"
		}
		log.StopWait()

		if err != nil {
			log.Fatalf("Unable to get supported languages: %s", err.Error())
		}

		cmd.chartGenerator.Language = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "What is the major programming language of your project?\nSupported languages: " + strings.Join(supportedLanguages, ", "),
			DefaultValue:           cmd.chartGenerator.Language,
			ValidationRegexPattern: "^(" + strings.Join(supportedLanguages, ")|(") + ")$",
		})
	}
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()

	if err != nil {
		log.Fatalf("Error while creating Helm chart and Dockerfile: %s", err.Error())
	}

	createdChartYaml := map[interface{}]interface{}{}
	createdChartValuesYaml := map[interface{}]interface{}{}

	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/Chart.yaml", &createdChartYaml)
	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/values.yaml", &createdChartValuesYaml)

	createdChartYaml["name"] = cmd.config.DevSpace.Release.Name
	/*TODO
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
	*/
	yamlutil.WriteYamlToFile(createdChartYaml, cmd.chartGenerator.Path+"/chart/Chart.yaml")
	yamlutil.WriteYamlToFile(createdChartValuesYaml, cmd.chartGenerator.Path+"/chart/values.yaml")
}
