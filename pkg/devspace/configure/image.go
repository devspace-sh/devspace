package configure

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/dockerfile"
	"mvdan.cc/sh/v3/expand"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/loft-util/pkg/command"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

const dockerHubHostname = "hub.docker.com"

// AddImage adds an image to the provided config
func (m *manager) AddImage(imageName, image, projectNamespace, dockerfile string) error {
	var (
		useDockerHub          = "Use " + dockerHubHostname
		useGithubRegistry     = "Use GitHub image registry"
		useOtherRegistry      = "Use other registry"
		skipRegistry          = "Skip Registry"
		registryDefaultOption = skipRegistry
		registryUsernameHint  = " => you are logged in as %s"
		rootLevelDockerfile   = "Use this existing Dockerfile: " + dockerfile
		differentDockerfile   = "Use a different Dockerfile (e.g. ./backend/Dockerfile)"
		subPathDockerfile     = "Use an existing Dockerfile within this project"
		customBuild           = "Use alternative build tool (e.g. jib, bazel)"
		skip                  = "Skip / I don't know"
		err                   error
	)

	imageConfig := &latest.Image{
		Image:      strings.ToLower(image),
		Dockerfile: dockerfile,
	}

	buildMethods := []string{subPathDockerfile}

	stat, err := os.Stat(imageConfig.Dockerfile)
	if err == nil && !stat.IsDir() {
		buildMethods = []string{rootLevelDockerfile, differentDockerfile}
	}

	buildMethod, err := m.log.Question(&survey.QuestionOptions{
		Question:     "How should DevSpace build the container image for this project?",
		DefaultValue: buildMethods[0],
		Options:      append(buildMethods, customBuild, skip),
	})
	if err != nil {
		return err
	}

	if buildMethod == customBuild {
		buildCommand, err := m.log.Question(&survey.QuestionOptions{
			Question: "Please enter your build command without the image (e.g. `gradle jib --image` => DevSpace will append the image name automatically)",
		})
		if err != nil {
			return err
		}

		imageConfig.Custom = &latest.CustomConfig{
			Command: buildCommand + " --tag=$(get_image --only=tag " + imageName + ")",
		}
	} else {
		if buildMethod != skip && buildMethod != rootLevelDockerfile {
			imageConfig.Dockerfile, err = m.log.Question(&survey.QuestionOptions{
				Question: "Please enter the path to this Dockerfile: (Enter to skip)",
				ValidationFunc: func(value string) error {
					if value == "" {
						return nil
					}

					stat, err := os.Stat(value)
					if err == nil && !stat.IsDir() {
						return nil
					}
					return errors.New(fmt.Sprintf("Dockerfile `%s` does not exist or is not a file", value))
				},
			})
			if err != nil {
				return err
			}

			if imageConfig.Dockerfile != "" {
				imageConfig.Context, err = m.log.Question(&survey.QuestionOptions{
					Question:     "What is the build context for building this image?",
					DefaultValue: path.Dir(imageConfig.Dockerfile) + "/",
					ValidationFunc: func(value string) error {
						stat, err := os.Stat(value)
						if err != nil && !stat.IsDir() {
							return errors.New("Context path does not exist or is not a directory")
						}
						return nil
					},
				})
				if err != nil {
					return err
				}
			} else {
				buildMethod = skip
			}
		}
	}

	if image == "" && buildMethod != skip {
		kubeClient, err := kubectl.NewDefaultClient()
		if err != nil {
			return err
		}

		// Get docker client
		dockerClient, err := m.factory.NewDockerClientWithMinikube(context.TODO(), kubeClient, true, m.log)
		if err != nil {
			return errors.Errorf("Cannot create docker client: %v", err)
		}

		// Check if user is logged into docker hub
		isLoggedIntoDockerHub := false
		authConfig, err := dockerClient.GetAuthConfig(context.TODO(), dockerHubHostname, true)
		if err == nil && authConfig.Username != "" {
			useDockerHub = useDockerHub + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			isLoggedIntoDockerHub = true
		}

		// Check if user is logged into GitHub
		isLoggedIntoGitHub := false
		authConfig, err = dockerClient.GetAuthConfig(context.TODO(), generator.GithubContainerRegistry, true)
		if err == nil && authConfig.Username != "" {
			useGithubRegistry = useGithubRegistry + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			isLoggedIntoGitHub = true
		}

		// Set registry select options according to logged in status of dockerhub and github
		registryOptions := []string{skipRegistry, useDockerHub, useGithubRegistry, useOtherRegistry}
		if isLoggedIntoGitHub {
			registryDefaultOption = useGithubRegistry
			registryOptions = []string{useGithubRegistry, useDockerHub, useOtherRegistry, skipRegistry}
		} else if isLoggedIntoDockerHub {
			registryDefaultOption = useDockerHub
			registryOptions = []string{useDockerHub, useGithubRegistry, useOtherRegistry, skipRegistry}
		}

		selectedRegistry, err := m.log.Question(&survey.QuestionOptions{
			Question:     "If you were to push any images, which container registry would you want to push to?",
			DefaultValue: registryDefaultOption,
			Options:      registryOptions,
		})
		if err != nil {
			return err
		}

		if selectedRegistry == skipRegistry {
			imageConfig.Image = "my-image-registry.tld/username" + "/" + imageName
		} else {
			registryHostname := ""
			if selectedRegistry == useDockerHub {
				registryHostname = dockerHubHostname
			} else if selectedRegistry == useGithubRegistry {
				registryHostname = generator.GithubContainerRegistry
			} else {
				registryHostname, err = m.log.Question(&survey.QuestionOptions{
					Question:     "Please provide the registry hostname without the image path (e.g. gcr.io, ghcr.io, ecr.io)",
					DefaultValue: "gcr.io",
				})
				if err != nil {
					return err
				}
			}

			registryUsername, err := m.addPullSecretConfig(dockerClient, strings.Trim(registryHostname+"/username/app", "/"))
			if err != nil {
				return err
			}

			if registryUsername == "" {
				registryUsername = "username"
			}

			if selectedRegistry == useDockerHub {
				imageConfig.Image = registryUsername + "/" + imageName
			} else {
				if projectNamespace == "" {
					projectNamespace = registryUsername
				}

				if regexp.MustCompile(`^(.+\.)?gcr.io$`).Match([]byte(registryHostname)) {
					projectNamespace = "project"
					project, err := command.Output(context.TODO(), "", expand.ListEnviron(os.Environ()...), "gcloud", "config", "get-value", "project")
					if err == nil {
						projectNamespace = strings.TrimSpace(string(project))
					}
				}

				imageConfig.Image = registryHostname + "/" + projectNamespace + "/" + imageName
			}
		}
	}

	if buildMethod == skip {
		imageConfig.Image = "username/app"
		imageConfig.Dockerfile = "./Dockerfile"
	}

	m.config.Images[imageName] = imageConfig
	return nil
}

func (m *manager) addPullSecretConfig(dockerClient docker.Client, image string) (string, error) {
	var err error
	image, _, err = dockerfile.GetStrippedDockerImageName(strings.ToLower(image))
	if err != nil {
		return "", err
	}

	registryHostname, err := pullsecrets.GetRegistryFromImageName(image)
	if err != nil {
		return "", err
	}

	registryHostnamePrintable := registryHostname
	if registryHostnamePrintable == "" {
		registryHostnamePrintable = dockerHubHostname
	}

	usernameQuestion := fmt.Sprintf("What is your username for %s? (optional, Enter to skip)", registryHostnamePrintable)
	passwordQuestion := fmt.Sprintf("What is your password for %s? (optional, Enter to skip)", registryHostnamePrintable)
	if strings.Contains(registryHostname, "ghcr.io") || strings.Contains(registryHostname, "github.com") {
		usernameQuestion = "What is your GitHub username? (optional, Enter to skip)"
		passwordQuestion = "Please enter a GitHub personal access token (optional, Enter to skip)"
	}

	registryUsername := ""
	registryPassword := ""
	retry := false

	m.log.WriteString(logrus.WarnLevel, "\n")

	for {
		m.log.Info("Checking registry authentication for " + registryHostnamePrintable + "...")
		authConfig, err := dockerClient.Login(context.TODO(), registryHostname, registryUsername, registryPassword, true, retry, retry)
		if err == nil && (authConfig.Username != "" || authConfig.Password != "") {
			registryUsername = authConfig.Username

			m.log.Donef("Great! You are authenticated with %s", registryHostnamePrintable)
			break
		}

		m.log.WriteString(logrus.WarnLevel, "\n")
		m.log.Warnf("Unable to find registry credentials for %s", registryHostnamePrintable)
		m.log.Warnf("Running `%s` for you to authenticate with the registry (optional)", strings.TrimSpace("docker login "+registryHostname))

		registryUsername, err = m.log.Question(&survey.QuestionOptions{
			Question:               usernameQuestion,
			ValidationRegexPattern: "^.*$",
		})
		if err != nil {
			return "", err
		}

		if registryUsername != "" {
			registryPassword, err = m.log.Question(&survey.QuestionOptions{
				Question:               passwordQuestion,
				ValidationRegexPattern: "^.*$",
				IsPassword:             true,
			})
			if err != nil {
				return "", err
			}
		}

		m.log.WriteString(logrus.WarnLevel, "\n")

		// Check if docker is running
		_, runErr := command.Output(context.TODO(), "", expand.ListEnviron(os.Environ()...), "docker", "version")

		// If Docker is available, ask if we should retry registry login
		if runErr == nil && registryUsername != "" {
			retry = true
		}

		if !retry {
			m.log.Warn("Skip validating image registry credentials.")
			m.log.Warn("You may ignore this warning. Pushing images to a registry is *not* required.")

			usernameVar := "REGISTRY_USERNAME"
			passwordVar := "REGISTRY_PASSWORD"

			m.config.PullSecrets = map[string]*latest.PullSecretConfig{
				encoding.Convert(registryHostname): {
					Registry: registryHostname,
					Username: fmt.Sprintf("${%s}", usernameVar),
					Password: fmt.Sprintf("${%s}", passwordVar),
				},
			}

			if m.config.Vars == nil {
				m.config.Vars = map[string]*latest.Variable{}
			}
			m.config.Vars[passwordVar] = &latest.Variable{
				Name:     passwordVar,
				Password: true,
			}

			m.localCache.SetVar(usernameVar, registryUsername)
			m.localCache.SetVar(passwordVar, registryPassword)

			break
		}
	}

	return registryUsername, nil
}
