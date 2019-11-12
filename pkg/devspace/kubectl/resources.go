package kubectl

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GroupVersionExist checks if the given group version exists
func (client *client) GroupVersionExist(groupVersion string, resourceList []*metav1.APIResourceList) bool {
	for _, resources := range resourceList {
		if resources.GroupVersion == groupVersion {
			return true
		}
	}

	return false
}

// ResourceExist checks if the given resource exists in the group version
func (client *client) ResourceExist(groupVersion, name string, resourceList []*metav1.APIResourceList) bool {
	for _, resources := range resourceList {
		if resources.GroupVersion == groupVersion {
			for _, resource := range resources.APIResources {
				if resource.Name == name {
					return true
				}
			}
		}
	}

	return false
}
