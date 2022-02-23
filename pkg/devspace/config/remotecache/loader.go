package remotecache

import (
	"context"
	"encoding/base64"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"github.com/loft-sh/devspace/pkg/util/encryption"
	"gopkg.in/yaml.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Loader is the interface for loading the cache
type Loader interface {
	Load(ctx context.Context, client kubectl.Client) (Cache, error)
}

// New generates a new generated config
func New(secretName string) Cache {
	return &RemoteCache{
		Vars:        make(map[string]string),
		Deployments: make(map[string]DeploymentCache),
		Data:        make(map[string]string),
		secretName:  secretName,
	}
}

// NewCacheLoader creates a new remote cache loader for the given DevSpace configuration name
func NewCacheLoader(devSpaceName string) Loader {
	return &cacheLoader{
		secretName: secretName(devSpaceName),
	}
}

type cacheLoader struct {
	secretName string
}

func (c *cacheLoader) Load(ctx context.Context, client kubectl.Client) (Cache, error) {
	secret, err := client.KubeClient().CoreV1().Secrets(client.Namespace()).Get(ctx, c.secretName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}

		return New(c.secretName), nil
	} else if secret.Data == nil || len(secret.Data["cache"]) == 0 {
		return New(c.secretName), nil
	}

	remoteCache := &RemoteCache{}
	err = yaml.Unmarshal(secret.Data["cache"], remoteCache)
	if err != nil {
		return nil, err
	}

	if remoteCache.Data == nil {
		remoteCache.Data = make(map[string]string)
	}
	if remoteCache.Deployments == nil {
		remoteCache.Deployments = make(map[string]DeploymentCache)
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

	remoteCache.secretName = c.secretName
	return remoteCache, nil
}

func secretName(devSpaceName string) string {
	return encoding.SafeConcatName("devspace", "cache", devSpaceName)
}
