package configure

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/generated"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/devspace/deploy"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/covexo/devspace/pkg/util/tar"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/russross/blackfriday"
	"github.com/skratchdot/open-golang/open"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/repo"
)

// AddPackage adds a helm dependency to specified deployment
func AddPackage(skipQuestion bool, appVersion, chartVersion, deployment string, args []string) error {
	config := configutil.GetConfig()
	if config.DevSpace.Deployments == nil || (len(*config.DevSpace.Deployments) != 1 && deployment == "") {
		return fmt.Errorf("Please specify the deployment via the -d flag")
	}

	var deploymentConfig *v1.DeploymentConfig
	for _, deployConfig := range *config.DevSpace.Deployments {
		if deployment == "" || deployment == *deployConfig.Name {
			if deployConfig.Helm == nil || deployConfig.Helm.ChartPath == nil {
				return fmt.Errorf("Selected deployment %s is not a valid helm deployment", *deployConfig.Name)
			}

			deploymentConfig = deployConfig
			break
		}
	}

	if deploymentConfig == nil {
		log.Fatalf("Deployment %s not found", deployment)
	}

	kubectl, err := kubectl.NewClient()
	if err != nil {
		log.Fatalf("Unable to create new kubectl client: %v", err)
	}

	helm, err := helmClient.NewClient(kubectl, log.GetInstance(), false)
	if err != nil {
		log.Fatalf("Error initializing helm client: %v", err)
	}

	if len(args) != 1 {
		helm.PrintAllAvailableCharts()
		os.Exit(0)
	}

	log.StartWait("Search Chart")
	repo, version, err := helm.SearchChart(args[0], chartVersion, appVersion)
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}

	log.Done("Chart found")
	chartPath, err := filepath.Abs(*deploymentConfig.Helm.ChartPath)
	if err != nil {
		log.Fatal(err)
	}
	packageName := version.GetName()

	requirementsFile := filepath.Join(chartPath, "requirements.yaml")
	_, err = os.Stat(requirementsFile)
	if os.IsNotExist(err) {
		entry := "dependencies:\n" +
			"- name: \"" + version.GetName() + "\"\n" +
			"  version: \"" + version.GetVersion() + "\"\n" +
			"  repository: \"" + repo.URL + "\"\n"

		err = ioutil.WriteFile(requirementsFile, []byte(entry), 0600)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		yamlContents := map[interface{}]interface{}{}
		err = yamlutil.ReadYamlFromFile(requirementsFile, yamlContents)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", requirementsFile, err)
		}

		dependenciesArr := []interface{}{}
		if dependencies, ok := yamlContents["dependencies"]; ok {
			dependenciesArr, ok = dependencies.([]interface{})
			if ok == false {
				log.Fatalf("Error parsing %s: Key dependencies is not an array", requirementsFile)
			}
		}

		for _, existingDependency := range dependenciesArr {
			existingDependencyMap, ok := existingDependency.(map[interface{}]interface{})

			if ok {
				existingDepName := existingDependencyMap["name"]

				if existingDepName == packageName {
					log.Fatalf("Dependency %s already exists", packageName)
				}
			}
		}

		dependenciesArr = append(dependenciesArr, map[interface{}]interface{}{
			"name":       packageName,
			"version":    version.GetVersion(),
			"repository": repo.URL,
		})
		yamlContents["dependencies"] = dependenciesArr

		err = yamlutil.WriteYamlToFile(yamlContents, requirementsFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.StartWait("Update chart dependencies")
	err = helm.UpdateDependencies(chartPath)
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}

	// Check if key already exists
	valuesYaml := filepath.Join(chartPath, "values.yaml")
	valuesYamlContents := map[interface{}]interface{}{}

	err = yamlutil.ReadYamlFromFile(valuesYaml, valuesYamlContents)
	if err != nil {
		log.Fatalf("Error parsing %s: %v", valuesYaml, err)
	}

	// get default config for package
	packageDefaults, hasPackageDefaultValues := packageDefaultMap[packageName]

	if _, ok := valuesYamlContents[packageName]; ok == false {
		f, err := os.OpenFile(valuesYaml, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		packageDefaultValues := "{}"
		if hasPackageDefaultValues && packageDefaults.values != "" {
			packageDefaultValues = packageDefaults.values
		}

		if _, err = f.WriteString(packageComment + packageName + ":" + packageDefaultValues); err != nil {
			log.Fatal(err)
		}
	}
	serviceLabelSelector := map[string]*string{}

	packageService := &v1.ServiceConfig{
		Name:          configutil.String(packageName),
		LabelSelector: &serviceLabelSelector,
	}

	if hasPackageDefaultValues && len(packageDefaults.serviceSelectors) > 0 {
		for key, value := range packageDefaults.serviceSelectors {
			serviceLabelSelector[key] = configutil.String(value)
		}
	} else {
		serviceLabelSelector["app"] = configutil.String(*deploymentConfig.Name + "-" + packageName)
	}

	_, sericeNotFoundErr := configutil.GetService(*packageService.Name)

	if sericeNotFoundErr != nil {
		err = configutil.AddService(packageService)
		if err != nil {
			log.Fatalf("Unable to add service to config: %v", err)
		}
	}

	err = configutil.SaveConfig()
	if err != nil {
		log.Fatalf("Unable to save config: %v", err)
	}

	log.Donef("Successfully added package %s, you can now modify the configuration in '%s/values.yaml'\n", packageName, chartPath)

	if skipQuestion == false {
		shouldShowReadme := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to open the package README to see configuration options? (yes|no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes|no)",
		})

		if shouldShowReadme == "yes" {
			if repo.URL == defaultStableRepoURL {
				open.Start("https://github.com/helm/charts/tree/master/stable/" + packageName)
			} else {
				showReadme(chartPath, version)
			}
		}

		shouldRedeploy := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Do you want to re-deploy your DevSpace with the added package? (yes|no)",
			DefaultValue:           "yes",
			ValidationRegexPattern: "^(yes|no)",
		})

		if shouldRedeploy == "yes" {
			redeployAferPackageChange(kubectl, deploymentConfig)
		}
	}

	return nil
}

func redeployAferPackageChange(kubectl *kubernetes.Clientset, deploymentConfig *v1.DeploymentConfig) {
	config := configutil.GetConfig()
	listOptions := metav1.ListOptions{}
	deploymentNamespace := *deploymentConfig.Namespace

	if deploymentNamespace == "" {
		var err error

		deploymentNamespace, err = configutil.GetDefaultNamespace(config)
		if err != nil {
			log.Fatal("Unable to retrieve default namespace: %v", err)
		}
	}

	// Load generatedConfig
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading generated.yaml: %v", err)
	}

	log.StartWait("Re-deploying DevSpace")

	existingClusterServices, clusterServiceErr := kubectl.Core().Services(deploymentNamespace).List(listOptions)
	if clusterServiceErr != nil {
		log.Warnf("Unable to list Kubernetes services: %v", clusterServiceErr)
	}

	err = deploy.All(kubectl, generatedConfig, true, true, log.GetInstance())
	log.StopWait()

	// Save generated config
	err = generated.SaveConfig(generatedConfig)
	if err != nil {
		log.Fatalf("Error saving generated config: %v", err)
	}

	if err != nil {
		log.Fatal(err)
	}
	log.Done("Successfully re-deployed DevSpace")

	if clusterServiceErr == nil {
		log.StartWait("Detecting package services")
		clusterServices, clusterServiceErr := kubectl.Core().Services(deploymentNamespace).List(listOptions)
		log.StopWait()

		if clusterServiceErr != nil {
			log.Warnf("Unable to list Kubernetes services: %v", clusterServiceErr)
		} else {
			indent := "     "
			serviceTableHeader := []string{
				indent,
				"Hostname",
				"Ports",
			}
			serviceTableContent := [][]string{}

		OUTER:
			for _, clusterService := range clusterServices.Items {
				for _, existingClusterService := range existingClusterServices.Items {
					if clusterService.GetName() == existingClusterService.GetName() {
						continue OUTER
					}
				}
				ports := []string{}

				for _, servicePort := range clusterService.Spec.Ports {
					ports = append(ports, strconv.Itoa(int(servicePort.Port)))
				}

				serviceTableContent = append(serviceTableContent, []string{
					indent,
					clusterService.GetName(),
					strings.Join(ports, ", "),
				})
			}

			if clusterServices.Size() > 0 {
				log.Info("The following services are now available within your DevSpace:\n")
				log.PrintTable(serviceTableHeader, serviceTableContent)
				log.Write("\n")
			}
		}
	}
}

// RemovePackage removes a helm dependency from a deployment
func RemovePackage(removeAll bool, deployment string, args []string, log log.Logger) error {
	config := configutil.GetConfig()
	if config.DevSpace.Deployments == nil || (len(*config.DevSpace.Deployments) != 1 && deployment == "") {
		return fmt.Errorf("Please specify the deployment via the -d flag")
	}

	var deploymentConfig *v1.DeploymentConfig
	for _, deployConfig := range *config.DevSpace.Deployments {
		if deployment == "" || deployment == *deployConfig.Name {
			if deployConfig.Helm == nil || deployConfig.Helm.ChartPath == nil {
				return fmt.Errorf("Selected deployment %s is not a valid helm deployment", *deployConfig.Name)
			}

			deploymentConfig = deployConfig
			break
		}
	}

	if deploymentConfig == nil {
		return fmt.Errorf("Deployment %s not found", deployment)
	}

	chartPath, err := filepath.Abs(*deploymentConfig.Helm.ChartPath)
	if err != nil {
		return err
	}

	if len(args) == 0 && removeAll == false {
		return errors.New("You need to specify a package name or the --all flag")
	}

	requirementsPath := filepath.Join(chartPath, "requirements.yaml")
	yamlContents := map[interface{}]interface{}{}

	err = yamlutil.ReadYamlFromFile(requirementsPath, yamlContents)
	if err != nil {
		return err
	}

	if dependencies, ok := yamlContents["dependencies"]; ok {
		dependenciesArr, ok := dependencies.([]interface{})
		if ok == false {
			log.Fatalf("Error parsing yaml: %v", dependencies)
		}

		if removeAll == false {
			for key, dependency := range dependenciesArr {
				dependencyMap, ok := dependency.(map[interface{}]interface{})
				if ok == false {
					log.Fatalf("Error parsing yaml: %v", dependencies)
				}

				if name, ok := dependencyMap["name"].(string); ok {
					if name == args[0] {
						chartVersion, ok := dependencyMap["version"].(string)

						if ok {
							subChartPath := filepath.Join(chartPath, "charts", name+"-"+chartVersion+".tgz")

							err = os.Remove(subChartPath)
							if err != nil {
								log.Warnf("Unable to delete package file: %s\nError: %v", subChartPath, err)
							}
						}

						dependenciesArr = append(dependenciesArr[:key], dependenciesArr[key+1:]...)
						yamlContents["dependencies"] = dependenciesArr

						err = rebuildDependencies(chartPath, yamlContents, log)
						if err != nil {
							return err
						}

						break
					}
				}
			}

			log.Donef("Successfully removed dependency %s", args[0])
			return nil
		}

		yamlContents["dependencies"] = []interface{}{}

		subChartPath := filepath.Join(chartPath, "charts")

		err = os.RemoveAll(subChartPath)
		if err != nil {
			log.Warnf("Unable to delete package folder: %s\nError: %v", subChartPath, err)
		}

		err = rebuildDependencies(chartPath, yamlContents, log)
		if err != nil {
			return err
		}

		log.Done("Successfully removed all dependencies")
		return nil
	}

	log.Done("No dependencies found")

	return nil
}

func rebuildDependencies(chartPath string, newYamlContents map[interface{}]interface{}, log log.Logger) error {
	err := yamlutil.WriteYamlToFile(newYamlContents, filepath.Join(chartPath, "requirements.yaml"))
	if err != nil {
		return err
	}

	// Rebuild dependencies
	kubectl, err := kubectl.NewClient()
	if err != nil {
		return fmt.Errorf("Unable to create new kubectl client: %v", err)
	}

	helm, err := helmClient.NewClient(kubectl, log, false)
	if err != nil {
		return fmt.Errorf("Error initializing helm client: %v", err)
	}

	log.StartWait("Update chart dependencies")
	err = helm.UpdateDependencies(chartPath)
	log.StopWait()

	if err != nil {
		return err
	}

	return nil
}

func showReadme(chartPath string, chartVersion *repo.ChartVersion) {
	content, err := tar.ExtractSingleFileToStringTarGz(filepath.Join(chartPath, "charts", chartVersion.GetName()+"-"+chartVersion.GetVersion()+".tgz"), chartVersion.GetName()+"/README.md")
	if err != nil {
		log.Fatal(err)
	}

	output := blackfriday.MarkdownCommon([]byte(content))
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "Readme.html"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.Write(output)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
	open.Start(f.Name())
}
