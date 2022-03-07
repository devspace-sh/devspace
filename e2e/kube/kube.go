package kube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewKubeHelper() (*KubeHelper, error) {
	kubeClient, err := kubectl.NewDefaultClient()
	if err != nil {
		return nil, err
	}

	return &KubeHelper{
		client: kubeClient,
	}, nil
}

type KubeHelper struct {
	client kubectl.Client
}

func (k *KubeHelper) Client() kubectl.Client {
	return k.client
}

func (k *KubeHelper) RawClient() kubernetes.Interface {
	return k.client.KubeClient()
}

func (k *KubeHelper) ExecByImageSelector(imageSelector, namespace string, command []string) (string, error) {
	targetOptions := targetselector.NewOptionsFromFlags("", "", []string{imageSelector}, namespace, "").
		WithTimeout(120).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2))

	globalTargetSelector := targetselector.NewTargetSelector(targetOptions)
	container, err := globalTargetSelector.SelectSingleContainer(context.TODO(), k.client, log.Discard)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := k.client.ExecBuffered(context.TODO(), container.Pod, container.Container.Name, command, nil)
	if err != nil {
		return "", fmt.Errorf("exec error: %v %s", err, string(stderr))
	}

	return string(stdout), nil
}

func (k *KubeHelper) ExecByContainer(labelSelector, containerName, namespace string, command []string) (string, error) {
	targetOptions := targetselector.NewOptionsFromFlags(containerName, labelSelector, nil, namespace, "").
		WithTimeout(120).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2))

	globalTargetSelector := targetselector.NewTargetSelector(targetOptions)
	container, err := globalTargetSelector.SelectSingleContainer(context.TODO(), k.client, log.Discard)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := k.client.ExecBuffered(context.TODO(), container.Pod, container.Container.Name, command, nil)
	if err != nil {
		return "", fmt.Errorf("exec error: %v %s", err, string(stderr))
	}

	return string(stdout), nil
}

func (k *KubeHelper) CreateNamespace(name string) (string, error) {
	name = strings.ToLower(name + "-" + randutil.GenerateRandomString(5))
	_, err := k.client.KubeClient().CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return "", err
	}

	return name, nil
}

func (k *KubeHelper) DeleteNamespace(name string) error {
	err := k.client.KubeClient().CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	return nil
}
