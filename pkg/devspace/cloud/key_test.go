package cloud

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"gotest.tools/assert"
)

func TestGetClusterKey(t *testing.T) {
	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// Delete temp folder after test
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	provider := Provider{}
	key, err := provider.GetClusterKey(&Cluster{})
	assert.NilError(t, err, "Error getting clusterKey from empty cluster")
	assert.Equal(t, "", key, "A key from an empty cluster should be empty")

	survey.SetNextAnswer("123456")
	provider.ClusterKey = map[int]string{}
	_, err = provider.GetClusterKey(&Cluster{Owner: &Owner{}})
	assert.Error(t, err, "verify key: get token: Provider has no key specified", "Error getting clusterKey from empty cluster")

	provider.ClusterKey[2] = "someCluster"
	_, err = provider.GetClusterKey(&Cluster{Owner: &Owner{}})
	assert.Error(t, err, "verify key: get token: Provider has no key specified", "Error getting clusterKey from empty cluster")
}
