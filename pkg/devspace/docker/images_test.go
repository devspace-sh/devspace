package docker

import (
	"context"
	"testing"
	
	"github.com/docker/docker/api/types/image"
	log "github.com/loft-sh/devspace/pkg/util/log/testing"
	"gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type deleteImageTestCase struct {
	name string
	
	deletedImageName string
	expectedResponse []image.DeleteResponse
	expectedErr      bool
}

func TestDeleteImage(t *testing.T) {
	testCases := []deleteImageTestCase{
		{
			name:             "Delete by name",
			deletedImageName: "deleteThis",
			expectedResponse: []image.DeleteResponse{
				{
					Deleted:  "deleteThis",
					Untagged: "deleteThis",
				},
			},
		},
	}
	
	for _, testCase := range testCases {
		var (
			response []image.DeleteResponse
			err      error
		)
		
		client := &client{
			APIClient: &fakeDockerClient{},
		}
		
		if testCase.deletedImageName != "" {
			response, err = client.DeleteImageByName(context.Background(), testCase.deletedImageName, &log.FakeLogger{})
		}
		
		if !testCase.expectedErr {
			assert.NilError(t, err, "Unexpected error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected error %v in testCase %s", err, testCase.name)
		}
		
		authsAsYaml, err := yaml.Marshal(response)
		assert.NilError(t, err, "Error parsing response to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedResponse)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(authsAsYaml), string(expectedAsYaml), "Unexpected response in testCase %s", testCase.name)
	}
}
