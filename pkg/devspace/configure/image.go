package configure

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	v1 "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/generator"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/util/imageselector"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

const dockerHubHostname = "hub.docker.com"
const githubContainerRegistry = "ghcr.io"
const noRegistryImage = "devspace"

// addImage adds an image to the provided config
func (m *manager) AddImage(imageName, image, dockerfile, contextPath string, dockerfileGenerator *generator.DockerfileGenerator) error {
	var (
		useDockerHub          = "Use " + dockerHubHostname
		useGithubRegistry     = "Use GitHub image registry"
		useOtherRegistry      = "Use other registry"
		registryDefaultOption = useDockerHub
		registryUsernameHint  = " => you are logged in as %s"
		providedDockerfile    = "Based on this existing Dockerfile: " + dockerfile
		differentDockerfile   = "Based on a different existing Dockerfile (e.g. ./backend/Dockerfile)"
		createNewDockerfile   = "Create a new Dockerfile for this project"
		subPathDockerfile     = "Based on an existing Dockerfile within in this project (e.g. ./backend/Dockerfile)"
		customBuild           = "Using a custom build process (e.g. jib, bazel)"
		err                   error
	)

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

	noImageProvided := image == ""

	if noImageProvided {
		authConfig, err := dockerClient.GetAuthConfig(dockerHubHostname, true)
		if err == nil && authConfig.Username != "" {
			useDockerHub = useDockerHub + fmt.Sprintf(registryUsernameHint, authConfig.Username)
			registryDefaultOption = useDockerHub
		}

		authConfig, err = dockerClient.GetAuthConfig(githubContainerRegistry, true)
		if err == nil && authConfig.Username != "" {
			useGithubRegistry = useGithubRegistry + fmt.Sprintf(registryUsernameHint, authConfig.Username)
		}

		registryOptions := []string{useDockerHub, useGithubRegistry, useOtherRegistry}
		selectedRegistry, err := m.log.Question(&survey.QuestionOptions{
			Question:     "Which registry do you want to use to push images to?",
			DefaultValue: registryDefaultOption,
			Options:      registryOptions,
		})
		if err != nil {
			return err
		}

		registryHostname := ""

		if selectedRegistry == useDockerHub {
			// nothing to do here
		} else if selectedRegistry == useGithubRegistry {
			registryHostname = githubContainerRegistry
		} else {
			registryHostname, err = m.log.Question(&survey.QuestionOptions{
				Question:     "Please provide the registry hostname (e.g. gcr.io, ghcr.io, ecr.io)",
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

		if selectedRegistry == useDockerHub {
			image = registryUsername + "/" + imageName
		} else {
			projectPath := registryUsername + "/project"
			if regexp.MustCompile("^(.+\\.)?gcr.io$").Match([]byte(registryHostname)) {
				project, err := exec.Command("gcloud", "config", "get-value", "project").Output()

				if err == nil {
					projectPath = strings.TrimSpace(string(project))
				}
			}

			image = registryHostname + "/" + projectPath + "/" + imageName
		}

		image, err = m.log.Question(&survey.QuestionOptions{
			Question:     "Please provide the full image to be pushed for this project",
			DefaultValue: image,
			ValidationFunc: func(value string) error {
				_, _, err = imageselector.GetStrippedDockerImageName(strings.ToLower(value))
				return err
			},
		})
		if err != nil {
			return err
		}
	} else {
		_, err := m.addPullSecretConfig(dockerClient, image)
		if err != nil {
			return err
		}
	}

	imageConfig := &latest.ImageConfig{
		Image:      strings.ToLower(image),
		Dockerfile: dockerfile,
		Build: &v1.BuildConfig{
			Disabled: true,
		},
	}

	buildMethods := []string{createNewDockerfile, subPathDockerfile}

	stat, err := os.Stat(imageConfig.Dockerfile)
	if err == nil && stat.IsDir() == false {
		buildMethods = []string{providedDockerfile, differentDockerfile}
	}

	buildMethod, err := m.log.Question(&survey.QuestionOptions{
		Question:     "How should DevSpace build this image?",
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

		imageConfig.Build = &v1.BuildConfig{
			Custom: &v1.CustomConfig{
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
						if err == nil && stat.IsDir() == false {
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
						if err != nil && stat.IsDir() == false {
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

		targets, err := helper.GetDockerfileTargets(imageConfig.Dockerfile)
		if err != nil {
			return err
		}

		var target string
		if len(targets) > 0 {
			targetNone := "[none] (build complete Dockerfile)"
			targets = append(targets, targetNone)
			target, err = m.log.Question(&survey.QuestionOptions{
				Question: "Which build stage (target) within your Dockerfile do you want to use for development?\n  Choose `build` for quickstart projects.",
				Options:  targets,
			})
			if err != nil {
				return err
			}

			if target != targetNone {
				imageConfig.Build = &latest.BuildConfig{
					Docker: &latest.DockerConfig{
						Options: &latest.BuildOptions{
							Target: target,
						},
					},
				}
			}
		}
	}

	m.config.Images[imageName] = imageConfig

	return nil
}

func (m *manager) addPullSecretConfig(dockerClient docker.Client, image string) (string, error) {
	var (
		usernameQuestion = ""
		passwordQuestion = ""
		err              error
	)

	image, _, err = imageselector.GetStrippedDockerImageName(strings.ToLower(image))
	if err != nil {
		return "", err
	}

	registryHostname, err := pullsecrets.GetRegistryFromImageName(image)
	if err != nil {
		return "", err
	}

	if registryHostname == "" {
		usernameQuestion = "What is your Docker Hub username?"
		passwordQuestion = "What is your Docker Hub password? (will only be used for `docker login`)"
	} else if strings.Contains(registryHostname, "ghcr.io") || strings.Contains(registryHostname, "github.com") {
		usernameQuestion = "What is your GitHub username?"
		passwordQuestion = "Please enter a GitHub personal access token (will only be used for `docker login`)"
	}

	registryUsername := ""
	registryPassword := ""

	for true {
		m.log.StartWait("Checking registry authentication")
		authConfig, err := dockerClient.Login(registryHostname, registryUsername, registryPassword, true, false, false)
		m.log.StopWait()
		if err == nil && (authConfig.Username != "" || authConfig.Password != "") {
			registryUsername = authConfig.Username
			break
		}

		m.log.Warnf("Unable to find registry credentials for %s", registryHostname)
		m.log.Warnf("Running `docker login %s` for you to authenticate with the registry. Make sure you have push permissions", registryHostname)

		registryUsername, err = m.log.Question(&survey.QuestionOptions{
			Question:               usernameQuestion,
			ValidationRegexPattern: "^.*$",
		})
		if err != nil {
			return "", err
		}

		registryPassword, err = m.log.Question(&survey.QuestionOptions{
			Question:               passwordQuestion,
			ValidationRegexPattern: "^.*$",
			IsPassword:             true,
		})
		if err != nil {
			return "", err
		}

		// Check if docker is running
		runErr := exec.Command("docker version").Run()
		if runErr != nil {
			m.log.Warn("Docker daemon is not running. Start Docker daemon or images will be built using kaniko inside Kubernetes.")

			usernameVar := "REGISTRY_USERNAME"
			passwordVar := "REGISTRY_PASSWORD"

			m.config.PullSecrets = []*latest.PullSecretConfig{
				{
					Registry: registryHostname,
					Username: fmt.Sprintf("${%s}", usernameVar),
					Password: fmt.Sprintf("${%s}", passwordVar),
				},
			}

			m.config.Vars = append(m.config.Vars, &v1.Variable{
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
