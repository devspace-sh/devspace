package loader

import (
	"context"
	"encoding/json"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SecretVarsKey = "vars"
)

// RestoreVarsFromSecret reads the previously saved vars from a secret in kubernetes
func RestoreVarsFromSecret(client kubectl.Client, secretName string) (map[string]string, bool, error) {
	secret, err := client.KubeClient().CoreV1().Secrets(client.Namespace()).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return nil, false, err
		}

		return map[string]string{}, false, nil
	} else if secret.Data == nil || len(secret.Data[SecretVarsKey]) == 0 {
		return map[string]string{}, false, nil
	}

	vars := map[string]string{}
	err = json.Unmarshal(secret.Data[SecretVarsKey], &vars)
	if err != nil {
		return nil, false, errors.Wrap(err, "unmarshal vars")
	}

	return vars, true, nil
}

// SaveVarsInSecret saves the given variables in the given secret with the kubernetes client
func SaveVarsInSecret(client kubectl.Client, vars map[string]string, secretName string) error {
	if vars == nil {
		vars = map[string]string{}
	}

	// marshal vars
	bytes, err := json.Marshal(vars)
	if err != nil {
		return err
	}

	secret, err := client.KubeClient().CoreV1().Secrets(client.Namespace()).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return err
		}

		_, err = client.KubeClient().CoreV1().Secrets(client.Namespace()).Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Data: map[string][]byte{
				SecretVarsKey: bytes,
			},
		}, metav1.CreateOptions{})
		return err
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	secret.Data[SecretVarsKey] = bytes
	_, err = client.KubeClient().CoreV1().Secrets(client.Namespace()).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return err
}
