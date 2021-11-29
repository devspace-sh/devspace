package util

import (
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ConvertMap(m map[string]string) (map[k8sv1.ResourceName]resource.Quantity, error) {
	if m == nil {
		return nil, nil
	}

	retMap := map[k8sv1.ResourceName]resource.Quantity{}
	for k, v := range m {
		pv, err := resource.ParseQuantity(v)
		if err != nil {
			return nil, errors.Wrapf(err, "parse kaniko pod resource quantity %s", k)
		}

		retMap[k8sv1.ResourceName(k)] = pv
	}

	return retMap, nil
}
