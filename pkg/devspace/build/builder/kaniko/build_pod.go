package kaniko

import (
	"fmt"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko/util"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"

	"github.com/docker/distribution/reference"
	"gopkg.in/yaml.v3"
	jsonyaml "sigs.k8s.io/yaml"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"

	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The kaniko init image that we use by default
const kanikoInitImage = "alpine"

// The kaniko build image we use by default
const kanikoBuildImage = "gcr.io/kaniko-project/executor:v1.20.1"

// The context path within the kaniko pod
const kanikoContextPath = "/context"

// The file the init container will wait for
const doneFile = "/tmp/done"

// DevspaceQuota is the quota name of the space quota in the devspace cloud
const devspaceQuota = "devspace-quota"

// DevspaceLimitRange is the limit range name of the space limit range in the devspace cloud
const devspaceLimitRange = "devspace-limit-range"

// The generateName string for the kaniko pod that we use by default
const podGenerateName = "devspace-build-kaniko-"

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

func (b *Builder) getBuildPod(ctx devspacecontext.Context, buildID string, options *types.ImageBuildOptions, dockerfilePath string) (*k8sv1.Pod, error) {
	kanikoOptions := b.helper.ImageConf.Kaniko

	registryURL, err := pullsecrets.GetRegistryFromImageName(b.FullImageName)
	if err != nil {
		return nil, err
	}

	pullSecretName := pullsecrets.GetRegistryAuthSecretName(registryURL)
	if b.PullSecretName != "" {
		pullSecretName = b.PullSecretName
	}

	kanikoImage := kanikoBuildImage
	if kanikoOptions.Image != "" {
		kanikoImage = kanikoOptions.Image
	}

	kanikoInitImage := kanikoInitImage
	if kanikoOptions.InitImage != "" {
		kanikoInitImage = kanikoOptions.InitImage
	}

	kanikoPodGenerateName := podGenerateName
	if kanikoOptions.GenerateName != "" {
		kanikoPodGenerateName = kanikoOptions.GenerateName
	}

	// additional options to pass to kaniko
	kanikoArgs := []string{
		"--dockerfile=" + kanikoContextPath + "/" + filepath.Base(dockerfilePath),
		"--context=dir://" + kanikoContextPath,
	}

	// specify destinations
	for _, tag := range b.helper.ImageTags {
		kanikoArgs = append(kanikoArgs, "--destination="+b.helper.ImageName+":"+tag)
	}

	// set target
	if options.Target != "" {
		kanikoArgs = append(kanikoArgs, "--target="+options.Target)
	}

	// set snapshot mode
	if kanikoOptions.SnapshotMode != "" {
		kanikoArgs = append(kanikoArgs, "--snapshotMode="+kanikoOptions.SnapshotMode)
	} else {
		kanikoArgs = append(kanikoArgs, "--snapshotMode=time")
	}

	// allow insecure registry
	if b.allowInsecureRegistry {
		kanikoArgs = append(kanikoArgs, "--insecure", "--skip-tls-verify")
	}

	// build args
	for key, value := range options.BuildArgs {
		newKanikoArg := fmt.Sprintf("%v=%v", key, *value)
		kanikoArgs = append(kanikoArgs, "--build-arg", newKanikoArg)
	}

	// cache flags
	if kanikoOptions.Cache {
		ref, err := reference.ParseNormalizedNamed(b.FullImageName)
		if err != nil {
			return nil, err
		}

		kanikoArgs = append(kanikoArgs, "--cache=true", "--cache-repo="+ref.Name())
	}

	// extra flags
	kanikoArgs = append(kanikoArgs, kanikoOptions.Args...)

	// build the volumes
	volumes := []k8sv1.Volume{
		{
			Name: "context",
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{},
			},
		},
	}
	volumeMounts := []k8sv1.VolumeMount{
		{
			Name:      "context",
			MountPath: kanikoContextPath,
		},
	}
	if !kanikoOptions.SkipPullSecretMount {
		volumes = append(volumes, k8sv1.Volume{
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
		})
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      pullSecretName,
			MountPath: "/kaniko/.docker",
		})
	}

	// add additional mounts
	for i, mount := range kanikoOptions.AdditionalMounts {
		volume := k8sv1.Volume{
			Name: fmt.Sprintf("additional-volume-%d", i),
		}

		// check which volume type we got
		if mount.Secret != nil {
			volume.VolumeSource = k8sv1.VolumeSource{
				Secret: &k8sv1.SecretVolumeSource{
					SecretName:  mount.Secret.Name,
					Items:       []k8sv1.KeyToPath{},
					DefaultMode: mount.Secret.DefaultMode,
				},
			}

			for _, item := range mount.Secret.Items {
				volume.VolumeSource.Secret.Items = append(volume.VolumeSource.Secret.Items, k8sv1.KeyToPath{
					Key:  item.Key,
					Path: item.Path,
					Mode: item.Mode,
				})
			}
		} else if mount.ConfigMap != nil {
			volume.VolumeSource = k8sv1.VolumeSource{
				ConfigMap: &k8sv1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{
						Name: mount.ConfigMap.Name,
					},
					Items:       []k8sv1.KeyToPath{},
					DefaultMode: mount.ConfigMap.DefaultMode,
				},
			}

			for _, item := range mount.ConfigMap.Items {
				volume.VolumeSource.ConfigMap.Items = append(volume.VolumeSource.ConfigMap.Items, k8sv1.KeyToPath{
					Key:  item.Key,
					Path: item.Path,
					Mode: item.Mode,
				})
			}
		} else {
			continue
		}

		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      volume.Name,
			ReadOnly:  mount.ReadOnly,
			MountPath: mount.MountPath,
			SubPath:   mount.SubPath,
		})
	}
	// create the build pod
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: kanikoPodGenerateName,
			Annotations:  map[string]string{},
			Labels: map[string]string{
				"devspace-build":    "true",
				"devspace-build-id": buildID,
				"devspace-pid":      ctx.RunID(),
			},
		},
		Spec: k8sv1.PodSpec{
			InitContainers: []k8sv1.Container{
				{
					Name:            "context",
					Image:           kanikoInitImage,
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
					Image:           kanikoImage,
					ImagePullPolicy: k8sv1.PullIfNotPresent,
					Command:         kanikoOptions.Command,
					Args:            kanikoArgs,
					VolumeMounts:    volumeMounts,
				},
			},
			NodeSelector:       kanikoOptions.NodeSelector,
			Tolerations:        kanikoOptions.Tolerations,
			ServiceAccountName: kanikoOptions.ServiceAccount,
			Volumes:            volumes,
			RestartPolicy:      k8sv1.RestartPolicyNever,
		},
	}

	// add extra annotations
	for k, v := range kanikoOptions.Annotations {
		pod.Annotations[k] = v
	}

	// add extra labels
	for k, v := range kanikoOptions.Labels {
		pod.Labels[k] = v
	}

	// add extra init env vars
	for k, v := range kanikoOptions.InitEnv {
		if len(pod.Spec.InitContainers[0].Env) == 0 {
			pod.Spec.InitContainers[0].Env = []k8sv1.EnvVar{}
		}

		pod.Spec.InitContainers[0].Env = append(pod.Spec.InitContainers[0].Env, k8sv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// add extra env vars
	for k, v := range kanikoOptions.Env {
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = []k8sv1.EnvVar{}
		}

		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, k8sv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	for k, v := range kanikoOptions.EnvFrom {
		if len(pod.Spec.Containers[0].Env) == 0 {
			pod.Spec.Containers[0].Env = []k8sv1.EnvVar{}
		}

		o, err := yaml.Marshal(v)
		if err != nil {
			return nil, errors.Errorf("error converting envFrom %s: %v", k, err)
		}

		source := &k8sv1.EnvVarSource{}
		err = jsonyaml.Unmarshal(o, source)
		if err != nil {
			return nil, errors.Errorf("error converting envFrom %s: %v", k, err)
		}

		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, k8sv1.EnvVar{
			Name:      k,
			ValueFrom: source,
		})
	}

	// check if we have specific options for the resources part
	if kanikoOptions.Resources == nil {
		// get available resources
		availableResources, err := b.getAvailableResources(ctx)
		if err != nil {
			return nil, err
		} else if availableResources != nil {
			limits := k8sv1.ResourceList{
				k8sv1.ResourceCPU:              availableResources.CPU,
				k8sv1.ResourceMemory:           availableResources.Memory,
				k8sv1.ResourceEphemeralStorage: availableResources.EphemeralStorage,
			}
			requests := k8sv1.ResourceList{
				k8sv1.ResourceCPU:              resource.MustParse("0"),
				k8sv1.ResourceMemory:           resource.MustParse("0"),
				k8sv1.ResourceEphemeralStorage: resource.MustParse("0"),
			}
			pod.Spec.InitContainers[0].Resources = k8sv1.ResourceRequirements{
				Limits:   limits,
				Requests: requests,
			}
			pod.Spec.Containers[0].Resources = k8sv1.ResourceRequirements{
				Limits:   limits,
				Requests: requests,
			}
		}
	} else {
		// convert resources
		limits, err := util.ConvertMap(kanikoOptions.Resources.Limits)
		if err != nil {
			return nil, errors.Wrap(err, "limits")
		}
		requests, err := util.ConvertMap(kanikoOptions.Resources.Requests)
		if err != nil {
			return nil, errors.Wrap(err, "requests")
		}

		pod.Spec.InitContainers[0].Resources = k8sv1.ResourceRequirements{
			Limits:   limits,
			Requests: requests,
		}
		pod.Spec.Containers[0].Resources = k8sv1.ResourceRequirements{
			Limits:   limits,
			Requests: requests,
		}
	}

	// return the build pod
	return pod, nil
}

// Determine available resources (This is only necessary in the devspace cloud)
func (b *Builder) getAvailableResources(ctx devspacecontext.Context) (*availableResources, error) {
	quota, err := ctx.KubeClient().KubeClient().CoreV1().ResourceQuotas(b.BuildNamespace).Get(ctx.Context(), devspaceQuota, metav1.GetOptions{})
	if err != nil {
		return nil, nil
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
	limitrange, err := ctx.KubeClient().KubeClient().CoreV1().LimitRanges(b.BuildNamespace).Get(ctx.Context(), devspaceLimitRange, metav1.GetOptions{})
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
