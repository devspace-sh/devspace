package podreplace

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/kaniko/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"path"
	"strings"
)

type containerPath struct {
	latest.PersistentPath
	Container string
}

func persistPaths(podName string, devPod *latest.DevPod, copiedPod *corev1.PodTemplateSpec) error {
	name := podName
	if devPod.PersistenceOptions != nil && devPod.PersistenceOptions.Name != "" {
		name = devPod.PersistenceOptions.Name
	}

	paths := []containerPath{}
	for _, p := range devPod.PersistPaths {
		paths = append(paths, containerPath{
			PersistentPath: p,
			Container:      devPod.Container,
		})
	}
	for _, c := range devPod.Containers {
		for _, p := range c.PersistPaths {
			paths = append(paths, containerPath{
				PersistentPath: p,
				Container:      c.Container,
			})
		}
	}
	if len(paths) == 0 {
		return nil
	}

	for i, p := range paths {
		if p.Path == "" {
			continue
		}

		subPath := p.VolumePath
		if subPath == "" {
			subPath = fmt.Sprintf("path-%d", i)
		}

		if len(copiedPod.Spec.Containers) > 1 && p.Container == "" {
			names := []string{}
			for _, c := range copiedPod.Spec.Containers {
				names = append(names, c.Name)
			}

			return fmt.Errorf("couldn't persist path %s as multiple containers were found %s, but no containerName was specified", p.Path, strings.Join(names, " "))
		}

		var container *corev1.Container
		for i, con := range copiedPod.Spec.Containers {
			if p.Container == "" || p.Container == con.Name {
				copiedPod.Spec.Containers[i].VolumeMounts = append(copiedPod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "devspace-persistence",
					MountPath: p.Path,
					SubPath:   subPath,
					ReadOnly:  p.ReadOnly,
				})

				container = &con
				break
			}
		}

		if container == nil || p.SkipPopulate || p.ReadOnly || (devPod.PersistenceOptions != nil && devPod.PersistenceOptions.ReadOnly) {
			continue
		}

		initContainer := corev1.Container{
			Name:    fmt.Sprintf("path-%d-init", i),
			Image:   container.Image,
			Command: []string{"sh"},
			Args:    []string{"-c", fmt.Sprintf("if [ ! -d \"/devspace-persistence/.devspace/\" ] && [ -d \"%s\" ]; then\n echo 'DevSpace is initializing the sync volume...'\n cp -a %s/. /devspace-persistence/ && mkdir /devspace-persistence/.devspace\nfi", path.Clean(p.Path), path.Clean(p.Path))},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "devspace-persistence",
					MountPath: "/devspace-persistence",
					SubPath:   subPath,
				},
			},
		}
		if p.InitContainer != nil && p.InitContainer.Resources != nil {
			// convert resources
			limits, err := util.ConvertMap(p.InitContainer.Resources.Limits)
			if err != nil {
				return errors.Wrap(err, "parse limits")
			}
			requests, err := util.ConvertMap(p.InitContainer.Resources.Requests)
			if err != nil {
				return errors.Wrap(err, "parse requests")
			}
			initContainer.Resources = corev1.ResourceRequirements{
				Limits:   limits,
				Requests: requests,
			}
		}

		// add an init container that pre-populates the persistent volume for that path
		copiedPod.Spec.InitContainers = append(copiedPod.Spec.InitContainers, initContainer)
	}

	copiedPod.Spec.Volumes = append(copiedPod.Spec.Volumes, corev1.Volume{
		Name: "devspace-persistence",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: name,
				ReadOnly:  devPod.PersistenceOptions != nil && devPod.PersistenceOptions.ReadOnly,
			},
		},
	})

	return nil
}
