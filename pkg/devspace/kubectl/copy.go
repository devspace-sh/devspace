package kubectl

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// CopyToContainer copies a local folder to a container path
func CopyToContainer(kubectl kubernetes.Interface, pod *k8sv1.Pod, container string, localPath, containerPath string, excludePaths []string) error {
	/// return copyToContainerTestable(Kubectl, Pod, Container, LocalPath, ContainerPath, ExcludePaths, false)

	return nil
}
