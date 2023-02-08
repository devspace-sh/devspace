package configure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/expand"

	"github.com/loft-sh/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
)

// AddKubectlDeployment adds a new kubectl deployment to the provided config
func (m *manager) AddKubectlDeployment(deploymentName string, isKustomization bool) error {
	question := "Please enter the paths to your Kubernetes manifests (comma separated, glob patterns are allowed, e.g. 'manifests/**' or 'kube/pod.yaml') [Enter to abort]"
	if isKustomization {
		question = "Please enter path to your Kustomization folder (e.g. ./kube/kustomization/)"
	}

	manifests, err := m.log.Question(&survey.QuestionOptions{
		Question: question,
		ValidationFunc: func(value string) error {
			if value == "" {
				return nil
			}

			if isKustomization {
				stat, err := os.Stat(path.Join(value, "kustomization.yaml"))
				if err == nil && !stat.IsDir() {
					return nil
				}
				return fmt.Errorf("path `%s` is not a Kustomization (kustomization.yaml missing)", value)
			} else {
				matches, err := filepath.Glob(value)
				if err != nil {
					return fmt.Errorf("path `%s` is not a valid glob pattern", value)
				}
				if len(matches) == 0 {
					return fmt.Errorf("path `%s` did not match any manifests", value)
				}
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	if manifests == "" {
		return fmt.Errorf("adding kubectl deployment aborted")
	}

	splitted := strings.Split(manifests, ",")
	splittedPointer := []string{}

	for _, s := range splitted {
		trimmed := strings.TrimSpace(s)
		splittedPointer = append(splittedPointer, trimmed)
	}

	if m.config.Deployments == nil {
		m.config.Deployments = map[string]*latest.DeploymentConfig{}
	}
	m.config.Deployments[deploymentName] = &latest.DeploymentConfig{
		Name: deploymentName,
		Kubectl: &latest.KubectlConfig{
			Manifests: splittedPointer,
		},
	}
	if isKustomization {
		m.config.Deployments[deploymentName].Kubectl.Kustomize = ptr.Bool(isKustomization)
	}
	m.isRemote[deploymentName] = false

	return nil
}

// AddHelmDeployment adds a new helm deployment to the provided config
func (m *manager) AddHelmDeployment(deploymentName string) error {
	for {
		helmConfig := &latest.HelmConfig{
			Chart: &latest.ChartConfig{},
			Values: map[string]interface{}{
				"someChartValue": "",
			},
		}

		var (
			localPath  = "Use a local Helm chart (e.g. ./helm/chart/)"
			chartRepo  = "Use a Helm chart repository (e.g. app-chart stored in https://charts.company.tld)"
			archiveURL = "Use a .tar.gz archive from URL (e.g. https://artifacts.company.tld/chart.tar.gz)"
			gitRepo    = "Use a chart from another git repository (e.g. you have an infra repo)"
			abort      = "Abort and return to more options"
		)
		chartLocation, err := m.log.Question(&survey.QuestionOptions{
			Question: "Which Helm chart do you want to use?",
			Options: []string{
				localPath,
				chartRepo,
				archiveURL,
				gitRepo,
				abort,
			},
		})
		if err != nil {
			return err
		}

		if chartLocation == abort {
			return errors.New("")
		}

		if chartLocation == localPath {
			localChartPath, err := m.log.Question(&survey.QuestionOptions{
				Question:               "Please enter the relative path to your local Helm chart (e.g. ./chart)",
				ValidationRegexPattern: ".+",
			})
			if err != nil {
				return err
			}

			absPath, err := filepath.Abs(".")
			if err != nil {
				return err
			}

			localChartPathRel, err := filepath.Rel(absPath, localChartPath)
			if err != nil {
				localChartPathRel = localChartPath
			}

			stat, err := os.Stat(path.Join(localChartPathRel, "Chart.yaml"))
			if err != nil || stat.IsDir() {
				m.log.WriteString(logrus.InfoLevel, "\n")
				m.log.Errorf("Local path `%s` is not a Helm chart (Chart.yaml missing)", localChartPathRel)
				continue
			}

			helmConfig.Chart.Name = localChartPathRel
			m.isRemote[deploymentName] = false
		} else if chartLocation == chartRepo || chartLocation == archiveURL {
		ChartRepoLoop:
			for {
				requestURL := ""

				if chartLocation == chartRepo {
					tempChartRepoURL, err := m.log.Question(&survey.QuestionOptions{
						Question: "Please specify the full URL of the chart repo (e.g. https://charts.org.tld/)",
						ValidationFunc: func(value string) error {
							_, err := url.ParseRequestURI(chartRepoURL(value))
							if err != nil {
								return err
							}
							return nil
						},
					})
					if err != nil {
						return err
					}

					helmConfig.Chart.RepoURL = chartRepoURL(tempChartRepoURL)

					requestURL = strings.TrimRight(helmConfig.Chart.RepoURL, "/") + "/index.yaml"

					helmConfig.Chart.Name, err = m.log.Question(&survey.QuestionOptions{
						Question:               "Please specify the name of the chart within your chart repository (e.g. payment-service)",
						ValidationRegexPattern: ".+",
					})
					if err != nil {
						return err
					}
				} else {
					requestURL, err = m.log.Question(&survey.QuestionOptions{
						Question:               "Please specify the full URL of your tar archived chart (e.g. https://artifacts.org.tld/chart.tar.gz)",
						ValidationRegexPattern: "^http(s)?://.*",
					})
					if err != nil {
						return err
					}

					helmConfig.Chart.Name = requestURL
				}

				username := ""
				password := ""

				for {
					httpClient := &http.Client{}
					req, err := http.NewRequest("GET", requestURL, nil)
					if err != nil {
						return err
					}

					if username != "" || password != "" {
						req.SetBasicAuth(username, password)
					}

					resp, err := httpClient.Do(req)
					if resp == nil {
						return err
					}

					if resp.StatusCode != http.StatusOK {
						if resp.StatusCode == http.StatusUnauthorized {
							m.log.Error("Not authorized to access Helm chart repository. Please provide auth credentials")

							username, err = m.log.Question(&survey.QuestionOptions{
								Question: "Enter your username for accessing " + requestURL,
							})
							if err != nil {
								return err
							}

							password, err = m.log.Question(&survey.QuestionOptions{
								Question: "Enter your password for accessing " + requestURL,
							})
							if err != nil {
								return err
							}
						} else {
							m.log.Errorf("Error: Received %s for chart repo index file `%s`", resp.Status, requestURL)
							break
						}
					} else {
						if username != "" || password != "" {
							usernameVar := "HELM_USERNAME"
							passwordVar := "HELM_PASSWORD"
							helmConfig.Chart.Username = fmt.Sprintf("${%s}", usernameVar)
							helmConfig.Chart.Password = fmt.Sprintf("${%s}", passwordVar)

							if m.config.Vars == nil {
								m.config.Vars = map[string]*latest.Variable{}
							}
							m.config.Vars[passwordVar] = &latest.Variable{
								Name:     passwordVar,
								Password: true,
							}

							m.localCache.SetVar(usernameVar, username)
							m.localCache.SetVar(passwordVar, password)
						}

						m.isRemote[deploymentName] = true
						break ChartRepoLoop
					}
				}
			}
		} else {
			for {
				chartTempPath := ".devspace/chart-repo"

				gitRepo, err := m.log.Question(&survey.QuestionOptions{
					Question: "Please specify the git repo that contains the chart (e.g. https://git.org.tld/team/project.git)",
				})
				if err != nil {
					return err
				}

				gitBranch, err := m.log.Question(&survey.QuestionOptions{
					Question:     "On which git branch is your Helm chart? (e.g. main, master, stable)",
					DefaultValue: "main",
				})
				if err != nil {
					return err
				}

				gitSubFolder, err := m.log.Question(&survey.QuestionOptions{
					Question: "In which folder is your Helm chart within this other git repo? (e.g. ./chart)",
				})
				if err != nil {
					return err
				}

				gitCommand := fmt.Sprintf("if [ -d '%s/.git' ]; then cd \"%s\" && git pull origin %s; else mkdir -p %s; git clone --single-branch --branch %s %s %s; fi", chartTempPath, chartTempPath, gitBranch, chartTempPath, gitBranch, gitRepo, chartTempPath)

				m.log.WriteString(logrus.InfoLevel, "\n")
				m.log.Infof("Cloning external repo `%s` containing to retrieve Helm chart", gitRepo)

				err = engine.ExecuteSimpleShellCommand(context.TODO(), "", expand.ListEnviron(os.Environ()...), os.Stdout, os.Stderr, nil, gitCommand)
				if err != nil {
					m.log.WriteString(logrus.InfoLevel, "\n")
					m.log.Errorf("Unable to clone repository `%s` (branch: %s)", gitRepo, gitBranch)
					continue
				}

				chartFolder := path.Join(chartTempPath, gitSubFolder)
				stat, err := os.Stat(chartFolder)
				if err != nil || !stat.IsDir() {
					m.log.WriteString(logrus.InfoLevel, "\n")
					m.log.Errorf("Local path `%s` does not exist or is not a directory", chartFolder)
					continue
				}

				helmConfig.Chart.Name = chartFolder
				m.config.Hooks = append(m.config.Hooks, &latest.HookConfig{
					Command: gitCommand,
					Events:  []string{"before:deploy"},
				})

				m.isRemote[deploymentName] = true
				break
			}
		}

		if m.config.Deployments == nil {
			m.config.Deployments = map[string]*latest.DeploymentConfig{}
		}
		m.config.Deployments[deploymentName] = &latest.DeploymentConfig{
			Name: deploymentName,
			Helm: helmConfig,
		}

		break
	}

	return nil
}

// AddTankaDeployment adds a new tanka deployment to the provided config
func (m *manager) AddTankaDeployment(deploymentName string) error {

	tankaPath, err := m.log.Question(&survey.QuestionOptions{
		Question:       "Please enter the path to your tanka root [Enter to abort]",
		ValidationFunc: func(value string) error {
			if value == "" {
				return nil
			}
			stat, err := os.Stat(value)
			if err != nil {
				return fmt.Errorf("path `%s` does not exist", value)
			}
			if !stat.IsDir() {
				return fmt.Errorf("path `%s` is not a directory", value)
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	if tankaPath == "" {
		return fmt.Errorf("adding tanka deployment aborted")
	}
	if m.config.Deployments == nil {
		m.config.Deployments = map[string]*latest.DeploymentConfig{}
	}

	environmentPath, err := m.log.Question(&survey.QuestionOptions{
		Question:       "Please enter Tanka's environment path (relative to Tanka's path) [Enter to abort]",
		ValidationFunc: func(value string) error {
			if value == "" {
				return nil
			}
			stat, err := os.Stat(path.Join(tankaPath, value))
			if err != nil {
				return fmt.Errorf("environment path `%s` does not exist", value)
			}
			if !stat.IsDir() {
				return fmt.Errorf("environment path `%s` is not a directory", value)
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	if environmentPath == "" {
		return fmt.Errorf("adding tanka deployment aborted")
	}

	m.config.Deployments[deploymentName] = &latest.DeploymentConfig{
		Name: deploymentName,
		Tanka: &latest.TankaConfig{
			Path:            tankaPath,
			EnvironmentPath: environmentPath,
		},
	}

	// TANKA TODO CHECK IF is Remote
	m.isRemote[deploymentName] = false

	return nil
}

// AddComponentDeployment adds a new deployment to the provided config
func (m *manager) AddComponentDeployment(deploymentName, image string, servicePort int) error {
	componentConfig := &latest.ComponentConfig{
		Containers: []*latest.ContainerConfig{
			{
				Image: image,
			},
		},
	}

	if servicePort > 0 {
		componentConfig.Service = &latest.ServiceConfig{
			Ports: []*latest.ServicePortConfig{
				{
					Port: &servicePort,
				},
			},
		}
	}

	chartValues, err := yamlutil.ToInterfaceMap(componentConfig)
	if err != nil {
		return err
	}

	if m.config.Deployments == nil {
		m.config.Deployments = map[string]*latest.DeploymentConfig{}
	}
	m.config.Deployments[deploymentName] = &latest.DeploymentConfig{
		Helm: &latest.HelmConfig{
			Chart: &latest.ChartConfig{
				Name:    helm.DevSpaceChartConfig.Name,
				RepoURL: helm.DevSpaceChartConfig.RepoURL,
			},
			Values: chartValues,
		},
	}
	m.isRemote[deploymentName] = true

	return nil
}

func (m *manager) IsRemoteDeployment(deploymentName string) bool {
	return m.isRemote[deploymentName]
}

func chartRepoURL(url string) string {
	repoURL := url
	if !(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) {
		repoURL = "https://" + url
	}
	return repoURL
}
