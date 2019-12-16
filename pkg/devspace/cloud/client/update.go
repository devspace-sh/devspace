package client

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"

	"github.com/pkg/errors"
)

// UseDefaultClusterDomain resets the used clusterID
func (c *client) UseDefaultClusterDomain(clusterID int, key string) (string, error) {
	output := struct {
		UseDefaultClusterDomain string `json:"manager_useDefaultClusterDomain"`
	}{}

	err := c.grapqhlRequest(`
		mutation($key:String!,$clusterID:Int!) {
			manager_useDefaultClusterDomain(key:$key,clusterID:$clusterID)
	  	}
	`, map[string]interface{}{
		"key":       key,
		"clusterID": clusterID,
	}, &output)
	if err != nil {
		return "", err
	}

	return output.UseDefaultClusterDomain, nil
}

// UpdateClusterDomain updates the domain of a cluster
func (c *client) UpdateClusterDomain(clusterID int, domain string) error {
	// Update cluster domain
	err := c.grapqhlRequest(`
		mutation ($clusterID:Int!, $domain:String!) {
			manager_updateClusterDomain(
				clusterID:$clusterID,
				domain:$domain,
				useSSL:false
			)
	  	}
	`, map[string]interface{}{
		"clusterID": clusterID,
		"domain":    domain,
	}, &struct {
		UpdateClusterDomain bool `json:"manager_updateClusterDomain"`
	}{})
	if err != nil {
		return errors.Wrap(err, "update cluster domain")
	}

	return nil
}

// DeployIngressController deploys an ingress controller for a cluster
func (c *client) DeployIngressController(clusterID int, key string, useHostNetwork bool) error {
	err := c.grapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!, $useHostNetwork:Boolean!) {
				manager_deployIngressController(
					clusterID:$clusterID,
					key:$key,
					useHostNetwork:$useHostNetwork
				)
			}
		`, map[string]interface{}{
		"clusterID":      clusterID,
		"key":            key,
		"useHostNetwork": useHostNetwork,
	}, &struct {
		Deploy bool `json:"manager_deployIngressController"`
	}{})
	if err != nil {
		return errors.Wrap(err, "deploy ingress controller")
	}

	return nil
}

// DeployAdmissionController deploys an admission controller for a cluster
func (c *client) DeployAdmissionController(clusterID int, key string) error {
	err := c.grapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_deployAdmissionController(
					clusterID:$clusterID,
					key:$key
				)
			}
		`, map[string]interface{}{
		"clusterID": clusterID,
		"key":       key,
	}, &struct {
		Deploy bool `json:"manager_deployAdmissionController"`
	}{})
	if err != nil {
		return errors.Wrap(err, "deploy admission controller")
	}

	return nil
}

// DeployGatekeeper deploys a gatekeeper for a cluster
func (c *client) DeployGatekeeper(clusterID int, key string) error {
	err := c.grapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_deployGatekeeper(clusterID: $clusterID, key: $key)
			}
		`, map[string]interface{}{
		"clusterID": clusterID,
		"key":       key,
	}, &struct {
		Deploy bool `json:"manager_deployGatekeeper"`
	}{})
	if err != nil {
		return errors.Wrap(err, "deploy gatekeeper")
	}

	return nil
}

// DeployGatekeeperRules deploys gatekeeper rules for a cluster
func (c *client) DeployGatekeeperRules(clusterID int, key string) error {
	err := c.grapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_updateGatekeeperRules(clusterID: $clusterID, key: $key, enableAll: true, forceDeploy: true)
			}
		`, map[string]interface{}{
		"clusterID": clusterID,
		"key":       key,
	}, &struct {
		Deploy bool `json:"manager_updateGatekeeperRules"`
	}{})
	if err != nil {
		return errors.Wrap(err, "deploy gatekeeper rules")
	}

	return nil
}

// DeployCertManager deploys a cert manager for a cluster
func (c *client) DeployCertManager(clusterID int, key string) error {
	err := c.grapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_deployCertManager(
					clusterID:$clusterID,
					key:$key
				)
			}
		`, map[string]interface{}{
		"clusterID": clusterID,
		"key":       key,
	}, &struct {
		Deploy bool `json:"manager_deployCertManager"`
	}{})
	if err != nil {
		return errors.Wrap(err, "deploy gatekeeper rules")
	}

	return nil
}

// InitCore initializes the core of a cluster
func (c *client) InitCore(clusterID int, key string, enablePodPolicy bool) error {
	err := c.grapqhlRequest(`
		mutation ($clusterID:Int!, $key:String!, $enablePodPolicy:Boolean!){
			manager_initializeCore(
				clusterID:$clusterID,
				key:$key,
				enablePodPolicy:$enablePodPolicy
			)
	  	}
	`, map[string]interface{}{
		"clusterID":       clusterID,
		"key":             key,
		"enablePodPolicy": enablePodPolicy,
	}, &struct {
		InitCore bool `json:"manager_initializeCore"`
	}{})
	if err != nil {
		return err
	}

	return nil
}

// UpdateUserClusterUser updates the user data of a cluster user
func (c *client) UpdateUserClusterUser(clusterUserID int, encryptedToken []byte) error {
	err := c.grapqhlRequest(`
		mutation($clusterUserID:Int!, $encryptedToken:String!) {
			manager_updateUserClusterUser(
				clusterUserID:$clusterUserID, 
				encryptedToken:$encryptedToken
			)
	  	}
	`, map[string]interface{}{
		"clusterUserID":  clusterUserID,
		"encryptedToken": encryptedToken,
	}, &struct {
		UpdateClusterUser bool `json:"manager_updateUserClusterUser"`
	}{})
	if err != nil {
		return err
	}

	return nil
}

// ResumeSpace resumes a space if its sleeping and sets the last activity to the current timestamp
func (c *client) ResumeSpace(spaceID int, key string, cluster *latest.Cluster) (bool, error) {
	// Do the request
	response := &struct {
		ResumeSpace bool `json:"manager_resumeSpace"`
	}{}
	err := c.grapqhlRequest(`
		mutation ($key:String, $spaceID: Int!){
			manager_resumeSpace(key: $key, spaceID: $spaceID)
		}
	`, map[string]interface{}{
		"key":     key,
		"spaceID": spaceID,
	}, response)
	if err != nil {
		return false, err
	}

	return response.ResumeSpace, nil
}
