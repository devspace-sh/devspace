package kaniko

import (
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"

	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/docker/distribution/reference"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The context path within the kaniko pod
const kanikoContextPath = "/context"

// The file the init container will wait for
const doneFile = "/tmp/done"

// DevspaceQuota is the quota name of the space quota in the devspace cloud
const devspaceQuota = "devspace-quota"

// DevspaceLimitRange is the limit range name of the space limit range in the devspace cloud
const devspaceLimitRange = "devspace-limit-range"

type availableResources struct {
	CPU              resource.Quantity
	Memory           resource.Quantity
	EphemeralStorage resource.Quantity
}

// The default resource limits to use for the kaniko pod
var defaultResources = &availableResources{
	CPU:              resource.MustParse("4"),
	Memory:           resource.MustParse("8Gi"),
	EphemeralStorage: resource.MustParse("10Gi"),
}

func (b *Builder) getBuildPod(buildID string, options *types.ImageBuildOptions, dockerfilePath string) (*k8sv1.Pod, error) {
	kanikoOptions := b.helper.ImageConf.Build.Kaniko

	registryURL, err := registry.GetRegistryFromImageName(b.FullImageName)
	if err != nil {
		return nil, err
	}

	pullSecretName := registry.GetRegistryAuthSecretName(registryURL)
	if b.PullSecretName != "" {
		pullSecretName = b.PullSecretName
	}

	// additional options to pass to kaniko
	kanikoArgs := []string{
		"--dockerfile=" + kanikoContextPath + "/" + filepath.Base(dockerfilePath),
		"--context=dir://" + kanikoContextPath,
		"--destination=" + b.FullImageName,
	}

	// Set snapshot mode
	if kanikoOptions.SnapshotMode != "" {
		kanikoArgs = append(kanikoArgs, "--snapshotMode="+kanikoOptions.SnapshotMode)
	} else {
		kanikoArgs = append(kanikoArgs, "--snapshotMode=time")
	}

	// Allow insecure registry
	if b.allowInsecureRegistry {
		kanikoArgs = append(kanikoArgs, "--insecure", "--skip-tls-verify")
	}

	// Build args
	for key, value := range options.BuildArgs {
		newKanikoArg := fmt.Sprintf("%v=%v", key, *value)
		kanikoArgs = append(kanikoArgs, "--build-arg", newKanikoArg)
	}

	// Extra flags
	if kanikoOptions.Flags != nil {
		for _, flag := range kanikoOptions.Flags {
			kanikoArgs = append(kanikoArgs, flag)
		}
	}

	// Cache
	if !options.NoCache {
		ref, err := reference.ParseNormalizedNamed(b.FullImageName)
		if err != nil {
			return nil, err
		}

		kanikoArgs = append(kanikoArgs, "--cache=true", "--cache-repo="+ref.Name())
	}

	// Get available resources
	availableResources, err := b.getAvailableResources()
	if err != nil {
		return nil, err
	}

	return &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "devspace-build-",
			Labels: map[string]string{
				"devspace-build":    "true",
				"devspace-build-id": buildID,
			},
		},
		Spec: k8sv1.PodSpec{
			InitContainers: []k8sv1.Container{
				{
					Name:            "context",
					Image:           "alpine",
					Command:         []string{"sh"},
					Args:            []string{"-c", "while [ ! -f " + doneFile + " ]; do sleep 2; done"},
					ImagePullPolicy: k8sv1.PullIfNotPresent,
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:      "context",
							MountPath: kanikoContextPath,
						},
					},
				},
			},
			Containers: []k8sv1.Container{
				{
					Name:            "kaniko",
					Image:           "gcr.io/kaniko-project/executor:v0.10.0",
					ImagePullPolicy: k8sv1.PullIfNotPresent,
					Args:            kanikoArgs,
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:      pullSecretName,
							MountPath: "/kaniko/.docker",
						},
						{
							Name:      "context",
							MountPath: kanikoContextPath,
						},
					},
					Resources: k8sv1.ResourceRequirements{
						Limits: k8sv1.ResourceList{
							k8sv1.ResourceCPU:              availableResources.CPU,
							k8sv1.ResourceMemory:           availableResources.Memory,
							k8sv1.ResourceEphemeralStorage: availableResources.EphemeralStorage,
						},
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceCPU:              resource.MustParse("0"),
							k8sv1.ResourceMemory:           resource.MustParse("0"),
							k8sv1.ResourceEphemeralStorage: resource.MustParse("0"),
						},
					},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: pullSecretName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: pullSecretName,
							Items: []k8sv1.KeyToPath{
								{
									Key:  k8sv1.DockerConfigJsonKey,
									Path: "config.json",
								},
							},
						},
					},
				},
				{
					Name: "context",
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{},
					},
				},
			},
			RestartPolicy: k8sv1.RestartPolicyNever,
		},
	}, nil
}

// Determine available resources (This is only necessary in the devspace cloud)
func (b *Builder) getAvailableResources() (*availableResources, error) {
	quota, err := b.helper.KubeClient.KubeClient().CoreV1().ResourceQuotas(b.BuildNamespace).Get(devspaceQuota, metav1.GetOptions{})
	if err != nil {
		return defaultResources, nil
	}

	availableResources := &availableResources{}

	// CPU
	availableResources.CPU, err = getAvailableResourceQuantity(defaultResources.CPU, k8sv1.ResourceLimitsCPU, quota)
	if err != nil {
		return nil, errors.Wrap(err, "get available resource quantity")
	}

	// Memory
	availableResources.Memory, err = getAvailableResourceQuantity(defaultResources.Memory, k8sv1.ResourceLimitsMemory, quota)
	if err != nil {
		return nil, errors.Wrap(err, "get available resource quantity")
	}

	// Ephemeral Storage
	availableResources.EphemeralStorage, err = getAvailableResourceQuantity(defaultResources.EphemeralStorage, k8sv1.ResourceLimitsEphemeralStorage, quota)
	if err != nil {
		return nil, errors.Wrap(err, "get available resource quantity")
	}

	// Get limitrange
	limitrange, err := b.helper.KubeClient.KubeClient().CoreV1().LimitRanges(b.BuildNamespace).Get(devspaceLimitRange, metav1.GetOptions{})
	if err != nil {
		return availableResources, nil
	}

	// Check if container limit is smaller than the available resources
	for _, limit := range limitrange.Spec.Limits {
		if limit.Type == k8sv1.LimitTypeContainer {
			if maxCPU, ok := limit.Max[k8sv1.ResourceCPU]; ok {
				if availableResources.CPU.Cmp(maxCPU) == 1 {
					availableResources.CPU = maxCPU
				}
			}
			if maxMemory, ok := limit.Max[k8sv1.ResourceMemory]; ok {
				if availableResources.Memory.Cmp(maxMemory) == 1 {
					availableResources.Memory = maxMemory
				}
			}
			if maxEphemeralStorage, ok := limit.Max[k8sv1.ResourceEphemeralStorage]; ok {
				if availableResources.EphemeralStorage.Cmp(maxEphemeralStorage) == 1 {
					availableResources.EphemeralStorage = maxEphemeralStorage
				}
			}
		}
	}

	return availableResources, nil
}

func getAvailableResourceQuantity(defaultQuantity resource.Quantity, resourceName k8sv1.ResourceName, quota *k8sv1.ResourceQuota) (resource.Quantity, error) {
	retLimit := defaultQuantity
	if quotaLimit, ok := quota.Status.Hard[resourceName]; ok {
		retLimit = quotaLimit
		if quotaUsed, ok := quota.Status.Used[resourceName]; ok {
			retLimit.Sub(quotaUsed)

			if retLimit.Cmp(defaultQuantity) == 1 {
				retLimit = defaultQuantity
			}
		}
	}

	// Check if limit == 0 or below zero
	if retLimit.Sign() != 1 {
		return resource.MustParse("0"), errors.Errorf("Available %s resource is zero or below zero: %s", resourceName, retLimit.String())
	}

	return retLimit, nil
}
