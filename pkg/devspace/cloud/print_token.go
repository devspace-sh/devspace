package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1alpha1"
)

// PrintToken prints and resumes a space if necessary
func (p *Provider) PrintToken(spaceID int) error {
	now := time.Now()

	// Check if token is cached
	if p.SpaceToken != nil && p.SpaceToken[spaceID] != nil {
		tokenCache := p.SpaceToken[spaceID]

		if now.Before(time.Unix(tokenCache.Expires, 0)) {
			// Check if we should resume
			if time.Unix(tokenCache.LastResume, 0).Add(time.Minute * 3).Before(now) {
				err := resume(p, tokenCache.Server, tokenCache.CaCert, tokenCache.Token, tokenCache.Namespace, spaceID, &Cluster{
					ClusterID:    tokenCache.ClusterID,
					Name:         tokenCache.ClusterName,
					EncryptToken: tokenCache.ClusterEncryptToken,
				})
				if err != nil {
					return errors.Wrap(err, "resume space")
				}

				tokenCache.LastResume = now.Unix()

				// We don't care so much if saving fails here
				_ = p.Save()
			}

			// Now print token
			return printToken(tokenCache.Token)
		}
	}

	// Token is not cached => hence we have to retrieve it
	space, err := p.GetSpace(spaceID)
	if err != nil {
		return fmt.Errorf("Error retrieving Spaces details: %v", err)
	}

	// Get service account
	serviceAccount, err := p.GetServiceAccount(space)
	if err != nil {
		return fmt.Errorf("Error retrieving space service account: %v", err)
	}

	// Save in cache
	if p.SpaceToken == nil {
		p.SpaceToken = map[int]*latest.SpaceToken{}
	}

	p.SpaceToken[spaceID] = &latest.SpaceToken{
		Token:     serviceAccount.Token,
		Namespace: serviceAccount.Namespace,
		Server:    serviceAccount.Server,
		CaCert:    serviceAccount.CaCert,

		ClusterID:           space.Cluster.ClusterID,
		ClusterName:         space.Cluster.Name,
		ClusterEncryptToken: space.Cluster.EncryptToken,

		LastResume: now.Unix(),
		Expires:    now.Add(time.Hour).Unix(),
	}

	// We don't care so much if saving fails here
	_ = p.Save()

	// Resume space
	err = resume(p, serviceAccount.Server, serviceAccount.CaCert, serviceAccount.Token, serviceAccount.Namespace, spaceID, space.Cluster)
	if err != nil {
		return errors.Wrap(err, "resume space")
	}

	// Print token and return
	return printToken(serviceAccount.Token)
}

func resume(p *Provider, server, caCert, token, namespace string, spaceID int, cluster *Cluster) error {
	// Resume space
	resumed, err := p.ResumeSpace(spaceID, cluster)
	if err != nil {
		return err
	}

	// We will wait a little bit till the space has resumed
	if resumed {
		// Give the controllers some time to create the pods
		time.Sleep(time.Second * 3)

		// // Load new kube config
		// config, err := kubeconfig.LoadNewConfig("devspace", server, caCert, token, namespace)
		// if err != nil {
		// 	return err
		// }

		// // Create kube client
		// client, err := kubectl.NewClientFromKubeConfig(config)
		// if err != nil {
		// 	return err
		// }

		// err = WaitForSpaceResume(client, namespace)
		// if err != nil {
		// 	return err
		// }
	}

	return nil
}

func printToken(token string) error {
	// Print token to stdout
	expireTime := metav1.NewTime(time.Now().Add(time.Hour))
	response := &v1alpha1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: "client.authentication.k8s.io/v1alpha1",
		},
		Status: &v1alpha1.ExecCredentialStatus{
			Token:               token,
			ExpirationTimestamp: &expireTime,
		},
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		return errors.Wrap(err, "json marshal")
	}

	_, err = os.Stdout.Write(bytes)
	return err
}
