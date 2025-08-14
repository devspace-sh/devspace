package remotecache

import (
	"context"
	"encoding/base64"
	"errors"
	"syscall"

	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/encryption"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SecretType = "devspace.sh/remote-cache"
)

// Loader is the interface for loading the cache
type Loader interface {
	Load(ctx context.Context, client kubectl.Client) (Cache, error)
}

// NewCache generates a new generated config
func NewCache(configName, secretName string) *RemoteCache {
	return &RemoteCache{
		Vars:        make(map[string]string),
		Deployments: []DeploymentCache{},
		DevPods:     []DevPodCache{},
		Data:        make(map[string]string),
		secretName:  secretName,
		configName:  configName,
	}
}

// NewCacheFromSecret loads the cache from secret
func NewCacheFromSecret(ctx context.Context, client kubectl.Client, secretName string) (*RemoteCache, error) {
	secret, err := client.KubeClient().CoreV1().Secrets(client.Namespace()).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// get config name
	configName := ""
	if secret.Labels != nil {
		configName = secret.Labels["name"]
	}

	// return secret
	if secret.Data == nil || len(secret.Data["cache"]) == 0 {
		s := NewCache(configName, secretName)
		s.secretNamespace = client.Namespace()
		return s, nil
	}

	remoteCache := &RemoteCache{}
	remoteCache.raw = secret.Data["cache"]
	err = yamlutil.Unmarshal(secret.Data["cache"], remoteCache)
	if err != nil {
		return nil, err
	}

	if remoteCache.Data == nil {
		remoteCache.Data = make(map[string]string)
	}
	if remoteCache.Vars == nil {
		remoteCache.Vars = make(map[string]string)
	}

	// Decrypt vars if necessary
	if remoteCache.VarsEncrypted {
		for k, v := range remoteCache.Vars {
			if len(v) == 0 {
				continue
			}

			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				// seems like not encrypted
				continue
			}

			decrypted, err := encryption.DecryptAES([]byte(localcache.EncryptionKey), decoded)
			if err != nil {
				// we cannot decrypt the variable, so we will ask the user again
				delete(remoteCache.Vars, k)
				continue
			}

			remoteCache.Vars[k] = string(decrypted)
		}

		remoteCache.VarsEncrypted = false
	}

	remoteCache.configName = configName
	remoteCache.secretName = secretName
	remoteCache.secretNamespace = client.Namespace()
	return remoteCache, nil
}

// NewCacheLoader creates a new remote cache loader for the given DevSpace configuration name
func NewCacheLoader(devSpaceName string) Loader {
	return &cacheLoader{
		secretName: secretName(devSpaceName),
		configName: devSpaceName,
	}
}

type cacheLoader struct {
	secretName string
	configName string
}

func (c *cacheLoader) Load(ctx context.Context, client kubectl.Client) (Cache, error) {
	remoteCache, err := NewCacheFromSecret(ctx, client, c.secretName)
	if err != nil {
		if !kerrors.IsNotFound(err) && !kerrors.IsForbidden(err) && !errors.Is(err, syscall.ECONNREFUSED) {
			return nil, err
		}

		s := NewCache(c.configName, c.secretName)
		s.secretNamespace = client.Namespace()
		return s, nil
	}

	return remoteCache, nil
}

func secretName(devSpaceName string) string {
	return encoding.SafeConcatName("devspace", "cache", devSpaceName)
}
