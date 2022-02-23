package remotecache

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/encryption"
	"github.com/loft-sh/devspace/pkg/util/patch"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sync"
)

type Cache interface {
	GetDeploymentCache(deploymentName string) (DeploymentCache, bool)
	DeleteDeploymentCache(deploymentName string)
	SetDeploymentCache(deploymentName string, deploymentCache DeploymentCache)

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

	Deployments map[string]DeploymentCache `yaml:"deployments,omitempty"`

	// Data is arbitrary key value cache
	Data map[string]string `yaml:"data,omitempty"`

	// config path is the path where the cache was loaded from
	secretName  string     `yaml:"-" json:"-"`
	accessMutex sync.Mutex `yaml:"-" json:"-"`
}

// DeploymentCache holds the information about a specific deployment
type DeploymentCache struct {
	DeploymentConfigHash string `yaml:"deploymentConfigHash,omitempty"`

	HelmRelease         string `yaml:"helmRelease,omitempty"`
	HelmOverridesHash   string `yaml:"helmOverridesHash,omitempty"`
	HelmChartHash       string `yaml:"helmChartHash,omitempty"`
	HelmValuesHash      string `yaml:"helmValuesHash,omitempty"`
	HelmReleaseRevision string `yaml:"helmReleaseRevision,omitempty"`

	KubectlManifestsHash string `yaml:"kubectlManifestsHash,omitempty"`
}

func (l *RemoteCache) GetDeploymentCache(deploymentName string) (DeploymentCache, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Deployments[deploymentName]
	return cache, ok
}

func (l *RemoteCache) DeleteDeploymentCache(deploymentName string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	delete(l.Deployments, deploymentName)
}

func (l *RemoteCache) SetDeploymentCache(deploymentName string, deploymentCache DeploymentCache) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Deployments[deploymentName] = deploymentCache
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
	if os.Getenv(localcache.DevSpaceDisableVarsEncryptionEnv) != "true" && localcache.EncryptionKey != "" {
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

	secret, err := client.KubeClient().CoreV1().Secrets(client.Namespace()).Get(ctx, l.secretName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrapf(err, "get cache secret")
		}

		_, err = client.KubeClient().CoreV1().Secrets(client.Namespace()).Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      l.secretName,
				Namespace: client.Namespace(),
			},
			Data: map[string][]byte{
				"cache": data,
			},
		}, metav1.CreateOptions{})
		return errors.Wrap(err, "create cache secret")
	}

	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	cacheData := secret.Data["cache"]
	if string(cacheData) == string(data) {
		return nil
	}

	originalSecret := secret.DeepCopy()
	secret.Data["cache"] = data

	// create patch
	p := patch.MergeFrom(originalSecret)
	bytes, err := p.Data(secret)
	if err != nil {
		return errors.Wrap(err, "create parent patch")
	}

	_, err = client.KubeClient().CoreV1().Secrets(client.Namespace()).Patch(ctx, l.secretName, p.Type(), bytes, metav1.PatchOptions{})
	return err
}
