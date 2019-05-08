package kaniko

import (
	"testing"
	"os"
	"io/ioutil"
	"time"

	"github.com/otiai10/copy"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"k8s.io/client-go/kubernetes/fake"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testNamespace = "test-kaniko-build"

func TestKanikoBuildWithEntrypointOverride(t *testing.T) {
	// @Florian

	// 1. Write test dockerfile and context to a temp folder
	dir, err := ioutil.TempDir("", "testDocker")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/kaniko", dir)

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 5. Delete temp files
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	// 2. Create kubectl client
	deployConfig := &latest.DeploymentConfig{
		Name: ptr.String("test-deployment"),
		Component: &latest.ComponentConfig{
			Containers: &[]*latest.ContainerConfig{
				{
					Image: ptr.String("nginx"),
				},
			},
			Service: &latest.ServiceConfig{
				Ports: &[]*latest.ServicePortConfig{
					{
						Port: ptr.Int(3000),
					},
				},
			},
		},
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveConfig: "default",
		Configs: map[string]*generated.CacheConfig{
			"default": &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"default": &generated.ImageCache{
						Tag: "1.15", // This will be appended to nginx during deploy
					},
				},
			},
		},
	}
	generated.InitDevSpaceConfig(generatedConfig, "default")

	namespace := "test-kaniko-build"
	imageName := "testimage"
	buildArgs := make(map[string]*string)
	buildArgsNoPush := "true"
	buildArgs["--no-push"] = &buildArgsNoPush
	imageConfig := &latest.ImageConfig{
		Build: &latest.BuildConfig{
			Kaniko: &latest.KanikoConfig{
				Namespace: &namespace,
				Options: &latest.BuildOptions{
					BuildArgs: &buildArgs,
				},
			},
		},
		Image: &imageName,
	}

	// Create the fake client.
	kubeClient := fake.NewSimpleClientset()

	dockerClient, err := docker.NewClient(testConfig, true)
	if err != nil {
		t.Fatalf("Error creating docker client: %v", err)
	}

	builder, err := NewBuilder(testConfig, dockerClient, kubeClient, "", imageConfig, "v1", true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating new kaniko builder: %v", err)
	}

	// 3. Create test namespace test-kaniko-build
	_, err = kubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		_, err = kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		})
		if err != nil {
			t.Fatalf("Error creating namespace: %v", err)
		}
	}
	//pod := k8sv1.Pod{}
	//kubeClient.Core().Pods(namespace).Create(&pod)
	go func(){
		buildPod, err := kubeClient.Core().Pods(namespace).Get("", metav1.GetOptions{})
		for err != nil{
			time.Sleep(1 * time.Millisecond)
			buildPod, err = kubeClient.Core().Pods(namespace).Get("", metav1.GetOptions{})
		}
		buildPod.Status.InitContainerStatuses = make([]k8sv1.ContainerStatus, 1)
		buildPod.Status.InitContainerStatuses[0] = k8sv1.ContainerStatus{
			State: k8sv1.ContainerState{
				Running: &k8sv1.ContainerStateRunning{

				},
			},
		}
		kubeClient.Core().Pods(namespace).Update(buildPod)
	}()
	

	// 4. Build image with kaniko, but don't push it (In kaniko options use "--no-push" as flag)
	entrypoint := make([]*string, 3)

	entrypoint0 := "go"
	entrypoint1 := "run"
	entrypoint2 := "main.go"
	entrypoint[0] = &entrypoint0
	entrypoint[1] = &entrypoint1
	entrypoint[2] = &entrypoint2
	err = builder.BuildImage(".", "Dockerfile", &entrypoint, log.GetInstance())
	if err != nil {
		t.Fatalf("Error building image: %v", err)
	}

	// 5. Delete test namespace
	err = kubeClient.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error deleting namespace: %v", err)
	}
}
