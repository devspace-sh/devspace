package cloud

import (
	"encoding/json"
	"os"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1alpha1"
)

// PrintToken prints and resumes a space if necessary
func (p *Provider) PrintToken(spaceID int) error {
	space, wasUpdated, err := p.GetAndUpdateSpaceCache(spaceID, false)
	if err != nil {
		return err
	}

	if wasUpdated == false && time.Unix(space.LastResume, 0).Add(time.Minute*3).Before(time.Now()) == false {
		err := printToken(space.ServiceAccount.Token)
		if err != nil {
			return err
		}

		// We exit here directly (not a very elegant way, but we do not want to send mixpanel stats every time which delays all other commands)
		os.Exit(0)
	}

	// Resume space
	err = resume(p, space.ServiceAccount.Server, space.ServiceAccount.CaCert, space.ServiceAccount.Token, space.ServiceAccount.Namespace, spaceID, space.Space.Cluster)
	if err != nil {
		return errors.Wrap(err, "resume space")
	}

	// Update when the space was last resumed
	p.Spaces[spaceID].LastResume = time.Now().Unix()

	// We don't care so much if saving fails here
	_ = p.Save()

	// Print token and return
	return printToken(space.ServiceAccount.Token)
}

func resume(p *Provider, server, caCert, token, namespace string, spaceID int, cluster *latest.Cluster) error {
	// Resume space
	resumed, err := p.ResumeSpace(spaceID, cluster)
	if err != nil {
		// We ignore the error here, because we don't want kubectl or other commands to fail if we have an outage
		// return err
	}

	// We will wait a little bit till the space has resumed
	if resumed {
		// Give the controllers some time to create the pods
		time.Sleep(time.Second * 3)
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
