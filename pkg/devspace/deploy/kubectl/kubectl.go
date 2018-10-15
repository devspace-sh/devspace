package kubectl

import (
	"path/filepath"

	"github.com/covexo/devspace/pkg/util/yamlutil"

	"k8s.io/client-go/kubernetes"
)

// Deploy deploys all specified manifests via kubectl apply and adds to the specified image names the corresponding tags
func Deploy(client *kubernetes.Clientset, namespace string, images []string, tags map[string]string, manifests []string) error {
	for _, pattern := range manifests {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}

		for _, file := range files {
			err = applyFile(client, file, namespace, images, tags)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func applyFile(client *kubernetes.Clientset, file string, namespace string, images []string, tags map[string]string) error {
	y := make(map[interface{}]interface{})
	yamlutil.ReadYamlFromFile(file, y)

	match := func(key, value string) bool {
		return false
	}

	replace := func(value string) string {
		return ""
	}

	Walk(y, match, replace)

	//changedManifest, err := yaml.Marshal(y)
	//if err != nil {
	//	return err
	//}

	return nil
}
