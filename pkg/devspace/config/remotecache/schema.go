package remotecache

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/env"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/encryption"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Cache interface {
	GetDeployment(deploymentName string) (DeploymentCache, bool)
	DeleteDeployment(deploymentName string)
	ListDeployments() []DeploymentCache
	SetDeployment(deploymentName string, deploymentCache DeploymentCache)

	GetDevPod(devPodName string) (DevPodCache, bool)
	DeleteDevPod(devPodName string)
	ListDevPods() []DevPodCache
	SetDevPod(devPodName string, devPodCache DevPodCache)

	GetData(key string) (string, bool)
	SetData(key, value string)

	GetVar(varName string) (string, bool)
	SetVar(varName, value string)

	DeepCopy() Cache

	// Save persists changes to file
	Save(ctx context.Context, client kubectl.Client) error
}

// RemoteCache specifies the runtime cache
type RemoteCache struct {
	Vars          map[string]string `yaml:"vars,omitempty"`
	VarsEncrypted bool              `yaml:"varsEncrypted,omitempty"`

	DevPods     []DevPodCache     `yaml:"devPods,omitempty"`
	Deployments []DeploymentCache `yaml:"deployments,omitempty"`

	// Data is arbitrary key value cache
	Data map[string]string `yaml:"data,omitempty"`

	// config path is the path where the cache was loaded from
	secretName      string     `yaml:"-" json:"-"`
	configName      string     `yaml:"-" json:"-"`
	secretNamespace string     `yaml:"-" json:"-"`
	raw             []byte     `yaml:"-" json:"-"`
	accessMutex     sync.Mutex `yaml:"-" json:"-"`
}

type DevPodCache struct {
	// Name is the name of the dev pod
	Name string `yaml:"name,omitempty"`

	// Projects are the top level projects that have started this dev configuration
	Projects []string `yaml:"projects,omitempty"`

	// Namespace is the namespace where the replace happened
	Namespace string `yaml:"namespace,omitempty"`

	// Deployment is the deployment that was created by DevSpace
	Deployment string `yaml:"deployment,omitempty"`

	// TargetKind is the kind of the original parent
	TargetKind string `yaml:"parentKind,omitempty"`

	// TargetName is the parent name of the original parent
	TargetName string `yaml:"parentName,omitempty"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	Name string `yaml:"name,omitempty"`

	// Projects are the top level projects that have deployed this deployment
	Projects []string `yaml:"projects,omitempty"`

	// DeploymentConfigHash is the deployment config hashed
	DeploymentConfigHash string `yaml:"deploymentConfigHash,omitempty"`

	// Helm holds the helm cache
	Helm *HelmCache `yaml:"helmCache,omitempty"`

	// Kubectl holds the kubectl cache
	Kubectl *KubectlCache `yaml:"kubectlCache,omitempty"`

	// Tanka holds the tanka cache
	Tanka *TankaCache `yaml:"tankaCache,omitempty"`
}

type HelmCache struct {
	Release          string `yaml:"release,omitempty"`
	ReleaseNamespace string `yaml:"releaseNamespace,omitempty"`

	OverridesHash   string `yaml:"overridesHash,omitempty"`
	ChartHash       string `yaml:"chartHash,omitempty"`
	ValuesHash      string `yaml:"valuesHash,omitempty"`
	ReleaseRevision string `yaml:"releaseRevision,omitempty"`
}

type KubectlCache struct {
	Objects       []KubectlObject `yaml:"kubectlObjects,omitempty"`
	ManifestsHash string          `yaml:"kubectlManifestsHash,omitempty"`
}

type TankaCache struct {
	AppliedTankaConfig *latest.TankaConfig `yaml:"appliedTankaConfig,omitempty"`
}

type KubectlObject struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
}

func (l *RemoteCache) ListDevPods() []DevPodCache {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	retArr := []DevPodCache{}
	retArr = append(retArr, l.DevPods...)
	return retArr
}

func (l *RemoteCache) GetDevPod(devPodName string) (DevPodCache, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	for _, dP := range l.DevPods {
		if dP.Name == devPodName {
			return dP, true
		}
	}
	return DevPodCache{
		Name: devPodName,
	}, false
}

func (l *RemoteCache) DeleteDevPod(devPodName string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	newArr := []DevPodCache{}
	for _, dP := range l.DevPods {
		if dP.Name == devPodName {
			continue
		}
		newArr = append(newArr, dP)
	}
	l.DevPods = newArr
}

func (l *RemoteCache) SetDevPod(devPodName string, devPodCache DevPodCache) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	for i, dP := range l.DevPods {
		if dP.Name == devPodName {
			l.DevPods[i] = devPodCache
			return
		}
	}
	l.DevPods = append(l.DevPods, devPodCache)
}

func (l *RemoteCache) ListDeployments() []DeploymentCache {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	retArr := []DeploymentCache{}
	retArr = append(retArr, l.Deployments...)
	return retArr
}

func (l *RemoteCache) GetDeployment(deploymentName string) (DeploymentCache, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	for _, dP := range l.Deployments {
		if dP.Name == deploymentName {
			return dP, true
		}
	}
	return DeploymentCache{
		Name: deploymentName,
	}, false
}

func (l *RemoteCache) DeleteDeployment(deploymentName string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	newArr := []DeploymentCache{}
	for _, dP := range l.Deployments {
		if dP.Name == deploymentName {
			continue
		}
		newArr = append(newArr, dP)
	}
	l.Deployments = newArr
}

func (l *RemoteCache) SetDeployment(deploymentName string, deploymentCache DeploymentCache) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	for i, dP := range l.Deployments {
		if dP.Name == deploymentName {
			l.Deployments[i] = deploymentCache
			return
		}
	}
	l.Deployments = append(l.Deployments, deploymentCache)
}

func (l *RemoteCache) GetData(key string) (string, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Data[key]
	return cache, ok
}

func (l *RemoteCache) SetData(key, value string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Data[key] = value
}

func (l *RemoteCache) GetVar(varName string) (string, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Vars[varName]
	return cache, ok
}

func (l *RemoteCache) SetVar(varName, value string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Vars[varName] = value
}

// DeepCopy creates a deep copy of the config
func (l *RemoteCache) DeepCopy() Cache {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	o, _ := yaml.Marshal(l)
	n := &RemoteCache{}
	_ = yaml.Unmarshal(o, n)
	n.secretName = l.secretName
	return n
}

// Save saves the config to the filesystem
func (l *RemoteCache) Save(ctx context.Context, client kubectl.Client) error {
	if l.secretName == "" {
		return fmt.Errorf("no secret specified where to save the remote cache")
	}

	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}

	copiedConfig := &RemoteCache{}
	err = yaml.Unmarshal(data, copiedConfig)
	if err != nil {
		return err
	}

	// encrypt variables
	if env.GlobalGetEnv(localcache.DevSpaceDisableVarsEncryptionEnv) != "true" && localcache.EncryptionKey != "" {
		for k, v := range copiedConfig.Vars {
			if len(v) == 0 {
				continue
			}

			encrypted, err := encryption.EncryptAES([]byte(localcache.EncryptionKey), []byte(v))
			if err != nil {
				return err
			}

			copiedConfig.Vars[k] = base64.StdEncoding.EncodeToString(encrypted)
		}

		copiedConfig.VarsEncrypted = true
	}

	// marshal again with the encrypted vars
	data, err = yaml.Marshal(copiedConfig)
	if err != nil {
		return err
	}
	if string(data) == string(l.raw) {
		return nil
	}

	namespace := l.secretNamespace
	if namespace == "" {
		namespace = client.Namespace()
	}

	waitErr := wait.PollImmediate(time.Second, time.Second*10, func() (done bool, err error) {
		secret, err := client.KubeClient().CoreV1().Secrets(namespace).Get(ctx, l.secretName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) && !kerrors.IsForbidden(err) {
				return false, errors.Wrapf(err, "get cache secret")
			}

			// don't do anything if its empty
			if len(l.Vars) == 0 && len(l.Data) == 0 && len(l.DevPods) == 0 && len(l.Deployments) == 0 {
				l.raw = data
				return true, nil
			}

			err = kubectl.EnsureNamespace(ctx, client, namespace, log.Discard)
			if err != nil {
				return false, err
			}

			_, err = client.KubeClient().CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      l.secretName,
					Namespace: client.Namespace(),
					Labels: map[string]string{
						"owner": "devspace",
						"name":  l.configName,
					},
				},
				Type: SecretType,
				Data: map[string][]byte{
					"cache": data,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				if kerrors.IsAlreadyExists(err) {
					return false, nil
				}

				return false, errors.Wrap(err, "create cache secret")
			}

			l.raw = data
			return true, nil
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		cacheData := secret.Data["cache"]
		if string(cacheData) == string(data) {
			l.raw = data
			return true, nil
		}

		secret.Data["cache"] = data

		// create patch
		_, err = client.KubeClient().CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				return false, nil
			}

			return false, err
		}

		l.raw = data
		return true, nil
	})
	return waitErr
}
