package configure

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

const dockerHubHostname = "hub.docker.com"
const githubContainerRegistry = "ghcr.io"

// AddImage adds an image to the provided config
func (m *manager) AddImage(imageName, image, dockerfile string, dockerfileGenerator *generator.DockerfileGenerator) error {
	var (
		useDockerHub          = "Use " + dockerHubHostname
		useGithubRegistry     = "Use GitHub image registry"
		useOtherRegistry      = "Use other registry"
		skipRegistry          = "Skip Registry"
		registryDefaultOption = skipRegistry
		registryUsernameHint  = " => you are logged in as %s"
		providedDockerfile    = "Based on this existing Dockerfile: " + dockerfile
		differentDockerfile   = "Based on a different existing Dockerfile (e.g. ./backend/Dockerfile)"
		createNewDockerfile   = "Create a new Dockerfile for this project"
		subPathDockerfile     = "Based on an existing Dockerfile within in this project (e.g. ./backend/Dockerfile)"
		customBuild           = "Using a custom build process (e.g. jib, bazel)"
		err                   error
	)

	imageConfig := &latest.ImageConfig{
		Image:      strings.ToLower(image),
		Dockerfile: dockerfile,
		Build: &latest.BuildConfig{
			Disabled: true,
		},
	}

	buildMethods := []string{createNewDockerfile, subPathDockerfile}

	stat, err := os.Stat(imageConfig.Dockerfile)
	if err == nil && !stat.IsDir() {
		buildMethods = []string{providedDockerfile, differentDockerfile}
	}

	buildMethod, err := m.log.Question(&survey.QuestionOptions{
		Question:     "How should DevSpace build the container image for this project?",
		DefaultValue: buildMethods[0],
		Options:      append(buildMethods, customBuild),
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

		buildCommandSplit := strings.Split(strings.TrimSpace(buildCommand), " ")

		imageConfig.Build = &latest.BuildConfig{
			Disabled: true,
			Custom: &latest.CustomConfig{
				Command: buildCommandSplit[0],
				Args:    buildCommandSplit[1:],
			},
		}
	} else {
		if buildMethod == createNewDockerfile {
			// Containerize application if necessary
			err = dockerfileGenerator.ContainerizeApplication(imageConfig.Dockerfile)
			if err != nil {
				return errors.Wrap(err, "containerize application")
			}
		} else {
			if buildMethod != providedDockerfile {
				imageConfig.Dockerfile, err = m.log.Question(&survey.QuestionOptions{
					Question: "Please enter the path to this Dockerfile",
					ValidationFunc: func(value string) error {
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
			}
		}
	}

	// Ignore error as context may not be a Space
	kubeContext, err := m.factory.NewKubeConfigLoader().GetCurrentContext()
	if err != nil {
		return err
	}

	// Get docker client
	dockerClient, err := m.factory.NewDockerClientWithMinikube(kubeContext, true, m.log)
	if err != nil {
		return errors.Errorf("Cannot create docker client: %v", err)
	}

	if image == "" {
		m.log.WriteString("\n")
		m.log.Info("DevSpace does *not* require pushing your images to a registry but let's assume you wanted to do that (optional)")

		registryOptions := []string{skipRegistry, useDockerHub, useGithubRegistry, useOtherRegistry}

		// Check if user is logged into docker hub
		isLoggedIntoDockerHub := false
		authConfig, err := dockerClient.GetAuthConfig(dockerHubHostname, true)
		if err == nil && authConfig.Username != "" {
			useDockerHub = useDockerHub + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			isLoggedIntoDockerHub = true
		}

		// Check if user is logged into GitHub
		isLoggedIntoGitHub := false
		authConfig, err = dockerClient.GetAuthConfig(githubContainerRegistry, true)
		if err == nil && authConfig.Username != "" {
			useGithubRegistry = useGithubRegistry + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			isLoggedIntoGitHub = true
		}

		// Set ansewer options according to logged in status of dockerhub and github
		if isLoggedIntoDockerHub {
			registryDefaultOption = useDockerHub
			registryOptions = []string{useDockerHub, useGithubRegistry, useOtherRegistry, skipRegistry}
		} else if isLoggedIntoGitHub {
			registryDefaultOption = useGithubRegistry
			registryOptions = []string{useGithubRegistry, useDockerHub, useOtherRegistry, skipRegistry}
		}

		selectedRegistry, err := m.log.Question(&survey.QuestionOptions{
			Question:     "Which registry would you want to use to push images to?",
			DefaultValue: registryDefaultOption,
			Options:      registryOptions,
		})
		if err != nil {
			return err
		}

		if selectedRegistry != skipRegistry {
			registryHostname := ""
			if selectedRegistry == useDockerHub {
				registryHostname = dockerHubHostname
			} else if selectedRegistry == useGithubRegistry {
				registryHostname = githubContainerRegistry
			} else {
				registryHostname, err = m.log.Question(&survey.QuestionOptions{
					Question:     "Please provide the registry hostname without the image path (e.g. gcr.io, ghcr.io, ecr.io)",
					DefaultValue: "gcr.io",
				})
				if err != nil {
					return err
				}
			}

			registryUsername, err := m.addPullSecretConfig(dockerClient, strings.Trim(registryHostname+"/test/test", "/"))
			if err != nil {
				return err
			}

			if registryUsername == "" {
				registryUsername = "username"
			}

			if selectedRegistry == useDockerHub {
				imageConfig.Image = registryUsername + "/" + imageName
			} else {
				projectPath := registryUsername + "/project"
				if regexp.MustCompile(`^(.+\.)?gcr.io$`).Match([]byte(registryHostname)) {
					project, err := exec.Command("gcloud", "config", "get-value", "project").Output()

					if err == nil {
						projectPath = strings.TrimSpace(string(project))
					}
				}

				imageConfig.Image = registryHostname + "/" + projectPath + "/" + imageName
			}
		} else {
			imageConfig.Image = "username" + "/" + imageName
		}
	} else {
		m.log.WriteString("\n")
		m.log.Info("DevSpace does *not* require pushing your images to a registry but let's check your registry credentials for this image (optional)")

		_, err := m.addPullSecretConfig(dockerClient, image)
		if err != nil {
			return err
		}
	}

	m.config.Images[imageName] = imageConfig

	return nil
}

func (m *manager) addPullSecretConfig(dockerClient docker.Client, image string) (string, error) {
	var err error
	image, _, err = imageselector.GetStrippedDockerImageName(strings.ToLower(image))
	if err != nil {
		return "", err
	}

	registryHostname, err := pullsecrets.GetRegistryFromImageName(image)
	if err != nil {
		return "", err
	}

	registryHostnamePrintable := registryHostname
	registryHostnameSpaced := " " + registryHostname

	if registryHostname == "" {
		registryHostnamePrintable = dockerHubHostname
		registryHostnameSpaced = registryHostname
	}

	usernameQuestion := fmt.Sprintf("What is your username for %s? (optional, Enter to skip)", registryHostnamePrintable)
	passwordQuestion := fmt.Sprintf("What is your password for %s? (optional, Enter to skip)", registryHostnamePrintable)

	if strings.Contains(registryHostname, "ghcr.io") || strings.Contains(registryHostname, "github.com") {
		usernameQuestion = "What is your GitHub username? (optional, Enter to skip)"
		passwordQuestion = "Please enter a GitHub personal access token (optional, Enter to skip)"
	}

	registryUsername := ""
	registryPassword := ""

	for {
		m.log.StartWait("Checking registry authentication for " + registryHostnamePrintable)
		authConfig, err := dockerClient.Login(registryHostname, registryUsername, registryPassword, true, false, false)
		m.log.StopWait()
		if err == nil && (authConfig.Username != "" || authConfig.Password != "") {
			registryUsername = authConfig.Username

			m.log.Donef("Great! You are authenticated with %s", registryHostnamePrintable)
			break
		}

		m.log.WriteString("\n")
		m.log.Warnf("Unable to find registry credentials for %s", registryHostnamePrintable)
		m.log.Warnf("Running `docker login%s` for you to authenticate with the registry (optional)", registryHostnameSpaced)

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

		m.log.WriteString("\n")

		// Check if docker is running
		runErr := exec.Command("docker", "version").Run()
		if runErr != nil || registryUsername == "" {
			if registryUsername == "" {
				m.log.Warn("Skipping image registry authentication.")
				m.log.Warn("You may ignore this warning. Pushing images to a registry is *not* required.")
			} else {
				m.log.Warn("Unable to verify image registry authentication because Docker daemon is not running.")
				m.log.Warn("You may ignore this warning. Pushing images to a registry is *not* required.")
			}

			usernameVar := "REGISTRY_USERNAME"
			passwordVar := "REGISTRY_PASSWORD"

			m.config.PullSecrets = []*latest.PullSecretConfig{
				{
					Registry: registryHostname,
					Username: fmt.Sprintf("${%s}", usernameVar),
					Password: fmt.Sprintf("${%s}", passwordVar),
				},
			}

			m.config.Vars = append(m.config.Vars, &latest.Variable{
				Name:     passwordVar,
				Password: true,
			})

			m.generated.Vars[usernameVar] = registryUsername
			m.generated.Vars[passwordVar] = registryPassword

			break
		}
	}

	return registryUsername, nil
}
