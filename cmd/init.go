package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/devspace/generator"
	"github.com/covexo/devspace/pkg/util/logutil"
	"github.com/covexo/devspace/pkg/util/randutil"
	"github.com/covexo/devspace/pkg/util/yamlutil"

	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/spf13/cobra"
)

type InitCmd struct {
	flags          *InitCmdFlags
	dsConfig       *v1.DevSpaceConfig
	privateConfig  *v1.PrivateConfig
	appConfig      *v1.AppConfig
	workdir        string
	chartGenerator *generator.ChartGenerator
}

type InitCmdFlags struct {
	reconfigure      bool
	overwrite        bool
	templateRepoURL  string
	templateRepoPath string
	language         string
}

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
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)

	cobraCmd.Flags().BoolVarP(&cmd.flags.reconfigure, "reconfigure", "r", cmd.flags.reconfigure, "Change existing configuration")
	cobraCmd.Flags().BoolVarP(&cmd.flags.overwrite, "overwrite", "o", cmd.flags.overwrite, "Overwrite existing chart files and Dockerfile")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoURL, "templateRepoUrl", cmd.flags.templateRepoURL, "Git repository for chart templates")
	cobraCmd.Flags().StringVar(&cmd.flags.templateRepoPath, "templateRepoPath", cmd.flags.templateRepoPath, "Local path for cloning chart template repository (uses temp folder if not specified)")
	cobraCmd.Flags().StringVarP(&cmd.flags.language, "language", "l", cmd.flags.language, "Programming language of your project")
}

func (cmd *InitCmd) Run(cobraCmd *cobra.Command, args []string) {
	log = logutil.GetLogger("default", true)
	workdir, workdirErr := os.Getwd()

	if workdirErr != nil {
		log.WithError(workdirErr).Panic("Unable to determine current workdir.")
	}
	cmd.workdir = workdir
	cmd.dsConfig = &v1.DevSpaceConfig{
		Version: "v1",
	}
	cmd.privateConfig = &v1.PrivateConfig{
		Version: "v1",
		Release: &v1.Release{
			Namespace: "default",
		},
		Cluster: &v1.Cluster{
			ApiServer: "https://192.168.99.100:8443",
			User:      &v1.User{},
		},
	}
	cmd.appConfig = &v1.AppConfig{
		Name: filepath.Base(cmd.workdir),
		Container: &v1.AppContainer{
			Ports: []int{},
		},
		External: &v1.AppExternal{
			Domain: "mydomain.com",
			Port:   80,
		},
	}
	dsConfigExists, _ := config.ConfigExists(cmd.dsConfig)
	privateConfigExists, _ := config.ConfigExists(cmd.privateConfig)

	if dsConfigExists || privateConfigExists {
		cmd.loadExistingConfig()
	}
	cmd.initChartGenerator()

	createChart := cmd.flags.overwrite

	if !cmd.flags.overwrite {
		_, dockerfileNotFound := os.Stat(cmd.chartGenerator.Path + "/Dockerfile")
		_, chartDirNotFound := os.Stat(cmd.chartGenerator.Path + "/chart")

		if dockerfileNotFound == nil || chartDirNotFound == nil {
			overwriteAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
				Question:               "Do you want to overwrite the Dockerfile and the existing files in /chart? (yes | no)",
				DefaultValue:           "no",
				ValidationRegexPattern: "^(yes)|(no)$",
			})
			createChart = (overwriteAnswer == "yes")
		} else {
			createChart = true
		}
	}

	if createChart {
		cmd.determineAppConfig()

		if cmd.privateConfig.Release == nil || len(cmd.privateConfig.Release.Name) == 0 {
			cmd.privateConfig.Release.Name = cmd.appConfig.Name
		}
		cmd.addPortForwarding()
		cmd.addSyncPath()
	}

	if cmd.flags.reconfigure || !dsConfigExists || !privateConfigExists {
		cmd.reconfigure()
	}

	if createChart {
		cmd.determineLanguage()
		cmd.createChart()
	}
}

func (cmd *InitCmd) loadExistingConfig() {
	config.LoadConfig(cmd.dsConfig)
	config.LoadConfig(cmd.privateConfig)
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
		existingChartYaml := map[interface{}]interface{}{}
		existingChartValuesYaml := map[interface{}]interface{}{}

		yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/Chart.yaml", existingChartYaml)
		yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/values.yaml", existingChartValuesYaml)

		cmd.appConfig.Name = existingChartYaml["name"].(string)

		applicationValues, applicationValuesCorrect := existingChartValuesYaml["container"].(map[interface{}]interface{})
		externalValues, externalValuesCorrect := existingChartValuesYaml["external"].(map[interface{}]interface{})

		if applicationValuesCorrect {
			value, isCorrect := applicationValues["port"].(int)

			if isCorrect {
				cmd.appConfig.Container.Ports = []int{value}
			}
		}

		if externalValuesCorrect {
			value, isCorrect := externalValues["domain"].(string)

			if isCorrect {
				cmd.appConfig.External.Domain = value
			}
		}
	}
	cmd.appConfig.Name = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "What is the name of your application?",
		DefaultValue:           cmd.appConfig.Name,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})

	// cmd.appConfig.Container.Ports, _ = strconv.Atoi(stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
	// 	Question:               "Which port(s) does your application listen on? (separated by spaces)",
	// 	DefaultValue:           strconv.Itoa(cmd.appConfig.Container.Port),
	// 	ValidationRegexPattern: "^[1-9][0-9]{0,4}?(\\s[1-9][0-9]{0,4})?$",
	// }))

	portsToSliceStr := []string{}

	for _, port := range cmd.appConfig.Container.Ports {
		portsToSliceStr = append(portsToSliceStr, strconv.Itoa(port))
	}

	portStrings := strings.Split(stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "Which port(s) does your application listen on? (separated by spaces)",
		DefaultValue:           strings.Join(portsToSliceStr, " "),
		ValidationRegexPattern: "^([1-9][0-9]{0,4})?(\\s[1-9][0-9]{0,4})*?$",
	}), " ")

	for _, port := range portStrings {
		portInt, _ := strconv.Atoi(port)
		cmd.appConfig.Container.Ports = append(cmd.appConfig.Container.Ports, portInt)
	}
	/* TODO
	cmd.appConfig.External.Domain = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "Which domain do you want to run your application on?",
		DefaultValue:           cmd.appConfig.External.Domain,
		ValidationRegexPattern: "^([a-z0-9]([a-z0-9-]{0,120}[a-z0-9])?\\.)+[a-z0-9]{2,}$",
	})*/
}

func (cmd *InitCmd) addPortForwarding() {
	portForwardingMissing := true

OUTER:
	for _, portForwarding := range cmd.dsConfig.PortForwarding {
		for _, portMapping := range portForwarding.PortMappings {
			for _, port := range cmd.appConfig.Container.Ports {
				if portMapping.RemotePort == port {
					portForwardingMissing = false
					break OUTER
				}
			}
		}
	}

	if portForwardingMissing {
		for _, port := range cmd.appConfig.Container.Ports {
			cmd.dsConfig.PortForwarding = append(cmd.dsConfig.PortForwarding, &v1.PortForwarding{
				PortMappings: []*v1.PortMapping{
					&v1.PortMapping{
						LocalPort:  port,
						RemotePort: port,
					},
				},
				ResourceType: "pod",
				LabelSelector: map[string]string{
					"release": cmd.privateConfig.Release.Name,
				},
			})
		}
	}
}

func (cmd *InitCmd) addSyncPath() {
	syncPathMissing := true

	for _, syncPath := range cmd.dsConfig.SyncPaths {
		if syncPath.LocalSubPath == "./" || syncPath.ContainerPath == "/app" {
			syncPathMissing = false
			break
		}
	}

	if syncPathMissing {
		cmd.dsConfig.SyncPaths = append(cmd.dsConfig.SyncPaths, &v1.SyncPath{
			ContainerPath: "/app",
			LocalSubPath:  "./",
			ResourceType:  "pod",
			LabelSelector: map[string]string{
				"release": cmd.privateConfig.Release.Name,
			},
		})
	}
}

func (cmd *InitCmd) reconfigure() {
	clusterConfig := cmd.privateConfig.Cluster

	if len(clusterConfig.TillerNamespace) == 0 {
		clusterConfig.TillerNamespace = cmd.privateConfig.Release.Namespace
	}
	cmd.privateConfig.Release.Namespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "Which Kubernetes namespace should your application run in?",
		DefaultValue:           cmd.privateConfig.Release.Namespace,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})
	clusterConfig.TillerNamespace = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "Which Kubernetes namespace should your tiller server run in?",
		DefaultValue:           clusterConfig.TillerNamespace,
		ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
	})
	kubeClusterConfig := &v1.Cluster{
		User: &v1.User{},
	}
	skipClusterConfig := false

	config.LoadClusterConfig(kubeClusterConfig, false)

	if len(kubeClusterConfig.ApiServer) != 0 && len(kubeClusterConfig.CaCert) != 0 && len(kubeClusterConfig.User.ClientCert) != 0 && len(kubeClusterConfig.User.ClientKey) != 0 {
		skipAnswer := stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "Do you want to use your existing $HOME/.kube/config for Kubernetes access? (yes | no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes)|(no)$",
		})
		skipClusterConfig = (skipAnswer == "yes")
	}

	if skipClusterConfig {
		clusterConfig.UseKubeConfig = true
	} else {
		clusterConfig.ApiServer = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is your Kubernetes API Server URL? (e.g. https://127.0.0.1:8443)",
			DefaultValue:           clusterConfig.ApiServer,
			ValidationRegexPattern: "^https?://[a-z0-9-.]{0,99}:[0-9]{1,5}$",
		})
		clusterConfig.CaCert = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is the CA Certificate of your API Server? (PEM)",
			DefaultValue:           clusterConfig.CaCert,
			InputTerminationString: "-----END CERTIFICATE-----",
		})
		clusterConfig.User.Username = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is your Kubernetes username?",
			DefaultValue:           clusterConfig.User.Username,
			ValidationRegexPattern: v1.Kubernetes.RegexPatterns.Name,
		})
		clusterConfig.User.ClientCert = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is your Kubernetes client certificate? (PEM)",
			DefaultValue:           clusterConfig.User.ClientCert,
			InputTerminationString: "-----END CERTIFICATE-----",
		})
		clusterConfig.User.ClientKey = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is your Kubernetes client key? (RSA, PEM)",
			DefaultValue:           clusterConfig.User.ClientKey,
			InputTerminationString: "-----END RSA PRIVATE KEY-----",
		})
	}
	cmd.reconfigureRegistry()

	dsConfigErr := config.SaveConfig(cmd.dsConfig)

	if dsConfigErr != nil {
		log.WithError(dsConfigErr).Panic("Config error")
	}
	privateConfigErr := config.SaveConfig(cmd.privateConfig)

	if privateConfigErr != nil {
		log.WithError(privateConfigErr).Panic("Config error")
	}
}

func (cmd *InitCmd) reconfigureRegistry() {
	registryConfig := cmd.privateConfig.Registry

	enableAutomaticBuilds := stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
		Question:               "Do you want to enable automatic Docker image building?",
		DefaultValue:           "yes",
		ValidationRegexPattern: "^(yes)|(no)$",
	})

	if enableAutomaticBuilds == "yes" {
		if registryConfig == nil {
			registryConfig = &v1.RegistryAccess{}
			cmd.privateConfig.Registry = registryConfig
		}
		registryConfig.Release = &v1.Release{
			Name:      "devspace-registry",
			Namespace: cmd.privateConfig.Release.Namespace,
		}
		registryConfig.User = &v1.RegistryUser{}

		if cmd.privateConfig.Cluster.User != nil && len(cmd.privateConfig.Cluster.User.Username) != 0 {
			registryConfig.User.Username = cmd.privateConfig.Cluster.User.Username
		}

		if len(registryConfig.User.Username) == 0 {
			randomUserSuffix, randErr := randutil.GenerateRandomString(5)

			if randErr != nil {
				log.WithError(randErr).Panic("Error creating random username")
			}
			registryConfig.User.Username = "user-" + randomUserSuffix
		}

		if len(registryConfig.User.Password) == 0 {
			randomPassword, randErr := randutil.GenerateRandomString(12)

			if randErr != nil {
				log.WithError(randErr).Panic("Error creating random password")
			}
			registryConfig.User.Password = randomPassword
		}

		if cmd.dsConfig.Registry == nil {
			cmd.dsConfig.Registry = map[interface{}]interface{}{}
		}
		secrets, registryHasSecrets := cmd.dsConfig.Registry["secrets"]

		if !registryHasSecrets {
			secrets = map[interface{}]interface{}{}
			cmd.dsConfig.Registry["secrets"] = secrets
		}
		secretMap, secretsIsMap := secrets.(map[interface{}]interface{})

		if secretsIsMap {
			_, registryHasSecretHtpasswd := secretMap["htpasswd"]

			if !registryHasSecretHtpasswd {
				secretMap["htpasswd"] = ""
			}
		}
	}
}

func (cmd *InitCmd) determineLanguage() {
	if len(cmd.flags.language) != 0 {
		if cmd.chartGenerator.IsSupportedLanguage(cmd.flags.language) {
			cmd.chartGenerator.Language = cmd.flags.language
		} else {
			fmt.Println("Language '" + cmd.flags.language + "' not supported yet. Please open an issue here: https://github.com/covexo/devspace/issues/new?title=Feature%20Request:%20Language%20%22" + cmd.flags.language + "%22")
		}
	}

	if len(cmd.chartGenerator.Language) == 0 {
		cmd.chartGenerator.Language, _ = cmd.chartGenerator.GetLanguage()
		supportedLanguages, langErr := cmd.chartGenerator.GetSupportedLanguages()

		if cmd.chartGenerator.Language == "" {
			cmd.chartGenerator.Language = "none"
		}

		if langErr != nil {
			log.WithError(langErr).Panic("Unable to get supported languages")
		}
		cmd.chartGenerator.Language = stdinutil.GetFromStdin(&stdinutil.GetFromStdin_params{
			Question:               "What is the major programming language of your project?\nSupported languages: " + strings.Join(supportedLanguages, ", "),
			DefaultValue:           cmd.chartGenerator.Language,
			ValidationRegexPattern: "^(" + strings.Join(supportedLanguages, ")|(") + ")$",
		})
	}
}

func (cmd *InitCmd) createChart() {
	err := cmd.chartGenerator.CreateChart()

	if err != nil {
		log.WithError(err).Panic("Error while creating Helm chart and Dockerfile:")
	}
	createdChartYaml := map[interface{}]interface{}{}
	createdChartValuesYaml := map[interface{}]interface{}{}

	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/Chart.yaml", &createdChartYaml)
	yamlutil.ReadYamlFromFile(cmd.chartGenerator.Path+"/chart/values.yaml", &createdChartValuesYaml)

	createdChartYaml["name"] = cmd.appConfig.Name

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
}
