package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
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

	configutil.Workdir = dir
	defer func() {
		configutil.Workdir = workDirBefore
	}()

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

	resetCmdObj.kubectl, err = kubectl.NewClient()
	if err != nil {
		t.Error(err)
	}
	resetCmdObj.deleteRelease()
	resetCmdObj.deleteRegistry()
	resetCmdObj.deleteTiller()

	upCmdObj := UpCmd{
		flags: UpFlagsDefault,
	}

	mockStdin("exit\\\\n")
	defer cleanUpMockedStdin()

	log.Logdir, err = ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(log.Logdir)

	upCmdObj.Run(nil, []string{})

	//TODO: Somehow stop all processes from the command above

	resetCmdObj.Run(nil, []string{})

	_, err = os.Stat(path.Join(dir, "Dockerfile"))
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
