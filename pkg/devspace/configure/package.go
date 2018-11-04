package configure

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	helmClient "github.com/covexo/devspace/pkg/devspace/helm"
	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"
	"github.com/covexo/devspace/pkg/util/tar"
	"github.com/covexo/devspace/pkg/util/yamlutil"
	"github.com/russross/blackfriday"
	"github.com/skratchdot/open-golang/open"
	"k8s.io/helm/pkg/repo"
)

// AddPackage adds a helm dependency to specified deployment
func AddPackage(skipQuestion bool, appVersion, chartVersion, deployment string, args []string, log log.Logger) (string, string, error) {
	packageName := args[0]
	config := configutil.GetConfig()
	if config.DevSpace.Deployments == nil || (len(*config.DevSpace.Deployments) != 1 && deployment == "") {
		return "", "", fmt.Errorf("Please specify the deployment via the -d flag")
	}

	var deploymentConfig *v1.DeploymentConfig
	for _, deployConfig := range *config.DevSpace.Deployments {
		if deployment == "" || deployment == *deployConfig.Name {
			if deployConfig.Helm == nil || deployConfig.Helm.ChartPath == nil {
				return "", "", fmt.Errorf("Selected deployment %s is not a valid helm deployment", *deployConfig.Name)
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

	helm, err := helmClient.NewClient(kubectl, log, false)
	if err != nil {
		log.Fatalf("Error initializing helm client: %v", err)
	}

	if len(args) != 1 {
		helm.PrintAllAvailableCharts()
		os.Exit(0)
	}

	log.StartWait("Search Chart")
	repo, version, err := helm.SearchChart(packageName, chartVersion, appVersion)
	log.StopWait()

	if err != nil {
		log.Fatal(err)
	}

	log.Done("Chart found")
	chartPath, err := filepath.Abs(*deploymentConfig.Helm.ChartPath)
	if err != nil {
		log.Fatal(err)
	}

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

		dependenciesArr = append(dependenciesArr, map[interface{}]interface{}{
			"name":       version.GetName(),
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

	if _, ok := valuesYamlContents[version.GetName()]; ok == false {
		f, err := os.OpenFile(valuesYaml, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
		if _, err = f.WriteString("\n# Here you can specify the subcharts values (for more information see: https://github.com/helm/helm/blob/master/docs/chart_template_guide/subcharts_and_globals.md#overriding-values-from-a-parent-chart)\n" + version.GetName() + ": {}\n"); err != nil {
			log.Fatal(err)
		}
	}

	err = configutil.AddService(&v1.ServiceConfig{
		Name: configutil.String(packageName),
		LabelSelector: &map[string]*string{
			"chart": configutil.String(packageName),
		},
	})
	if err != nil {
		log.Fatalf("Unable to add service to config: %v", err)
	}

	err = configutil.SaveConfig()
	if err != nil {
		log.Fatalf("Unable to save config: %v", err)
	}

	if skipQuestion == false {
		showReadme(chartPath, version)
	}

	return version.GetName(), *deploymentConfig.Helm.ChartPath, nil
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

				if name, ok := dependencyMap["name"]; ok {
					if name == args[0] {
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
	showReadme := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Do you want to open the package README? (y|n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)",
	})

	if showReadme == "n" {
		return
	}

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
