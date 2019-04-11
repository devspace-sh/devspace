package cloud

import (
	"errors"
)

// CreateSpace creates a new space and returns the space id
func (p *Provider) CreateSpace(name string, projectID int, clusterID *int) (int, error) {

	// Response struct
	response := struct {
		CreateSpace *struct {
			SpaceID int
		} `json:"manager_createSpace"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
		mutation($spaceName: String!, $clusterID: Int, $projectID: Int!) {
			manager_createSpace(spaceName: $spaceName, clusterID: $clusterID, projectID: $projectID) {
				SpaceID
			}
		}
	`, map[string]interface{}{
		"spaceName": name,
		"projectID": projectID,
		"clusterID": clusterID,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateSpace == nil {
		return 0, errors.New("Couldn't create project: returned answer is null")
	}

	return response.CreateSpace.SpaceID, nil
}

// CreateProject creates a new project and returns the project id
func (p *Provider) CreateProject(projectName string) (int, error) {
	// Response struct
	response := struct {
		CreateProject *struct {
			ProjectID int
		} `json:"manager_createProject"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
		mutation($projectName: String!) {
			manager_createProject(projectName: $projectName) {
				ProjectID
			}
		}
	`, map[string]interface{}{
		"projectName": projectName,
	}, &response)
	if err != nil {
		return 0, err
	}

	// Check result
	if response.CreateProject == nil {
		return 0, errors.New("Couldn't create project: returned answer is null")
	}

	return response.CreateProject.ProjectID, nil
}
