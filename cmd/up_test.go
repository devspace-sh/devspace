package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/covexo/devspace/pkg/devspace/kubectl"
	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRun(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	err = fsutil.Copy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up"), dir, true)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(dir)

	workDirBefore, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	defer os.Chdir(workDirBefore)
	os.Chdir(dir)

	upCmdObj := UpCmd{
		flags: &UpCmdFlags{
			sync: false,
		},
	}

	mockStdin("exit\\\\n")
	defer cleanUpMockedStdin()

	defer func() {
		client, err := kubectl.NewClient()
		if err != nil {
			t.Error(err)
		}
		propagationPolicy := metav1.DeletePropagationForeground
		client.Core().Namespaces().Delete("test-cmd-up", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}()

	/*resetCmdObj := ResetCmd{
		flags: &ResetCmdFlags{},
	}
	resetCmdObj.kubectl, err = kubectl.NewClient()
	if err != nil {
		t.Error(err)
	}
	resetCmdObj.helm, err = helm.NewClient(resetCmdObj.kubectl, false)
	if err != nil {
		t.Error(err)
	}
	resetCmdObj.deleteRegistry()*/

	upCmdObj.Run(nil, []string{})
	log.StopFileLogging()

	//TODO: Somehow stop all processes from the command above

	testReset(t, dir)

}

func testReset(t *testing.T, dir string) {
	resetCmdObj := ResetCmd{
		flags: &ResetCmdFlags{
			deleteDockerfile:         true,
			deleteDockerignore:       true,
			deleteChart:              true,
			deleteRegistry:           true,
			deleteTiller:             true,
			deleteDevspaceFolder:     true,
			deleteRelease:            true,
			deleteClusterRoleBinding: false,
		},
	}
	resetCmdObj.Run(nil, []string{})

	_, err := os.Stat(path.Join(dir, "Dockerfile"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".dockerignore"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".devspace"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, "chart"))
	assert.Equal(t, true, os.IsNotExist(err))

	_, err = os.Stat(path.Join(dir, "testJavaScript.js"))
	assert.Equal(t, false, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, "package.json"))
	assert.Equal(t, false, os.IsNotExist(err))
}

var tmpfile *os.File
var oldStdin *os.File

func mockStdin(inputString string) error {
	//Code from https://stackoverflow.com/a/46365584 (modified)
	input := []byte(inputString)
	var err error
	tmpfile, err = ioutil.TempFile("", "testGetFromStdin")
	if err != nil {
		return errors.Trace(err)
	}

	if _, err := tmpfile.Write(input); err != nil {
		return errors.Trace(err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		return errors.Trace(err)
	}

	oldStdin = os.Stdin
	os.Stdin = tmpfile

	return nil
}

func cleanUpMockedStdin() {
	os.Remove(tmpfile.Name())
	os.Stdin = oldStdin
}
