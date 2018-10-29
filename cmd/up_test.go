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

func TestUpWithInternalRegistry(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseInternalRegistry"), t)
	defer resetWorkDir()

	upCmdObj := UpCmd{
		flags: UpFlagsDefault,
	}
	upCmdObj.flags.sync = false

	mockStdin("exit\\\\n")
	defer cleanUpMockedStdin()

	defer func() {
		client, err := kubectl.NewClient()
		if err != nil {
			t.Error(err)
		}
		propagationPolicy := metav1.DeletePropagationForeground
		client.Core().Namespaces().Delete("test-cmd-up-private-registry", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}()

	log.Debug("WorkDir=" + os.Getwd())
	upCmdObj.Run(nil, []string{})
	log.StopFileLogging()

	testReset(t)

}

/*func TestUpWithDockerHub(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseDockerHub"), t)
	defer resetWorkDir()

	upCmdObj := UpCmd{
		flags: &UpCmdFlags{},
	}
	upCmdObj.flags.sync = false

	mockStdin("exit\\\\n")
	defer cleanUpMockedStdin()

	defer func() {
		client, err := kubectl.NewClient()
		if err != nil {
			t.Error(err)
		}
		propagationPolicy := metav1.DeletePropagationForeground
		client.Core().Namespaces().Delete("217b737767c3420e68e6c3b659eb46bb", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}()

	upCmdObj.Run(nil, []string{})
	log.StopFileLogging()

	testReset(t, dir)

}*/

var workDirBefore string

func createTempFolderCopy(source string, t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	log.Debug("TempDir at " + dir)
	err = fsutil.Copy(source, dir, true)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(dir)

	workDirBefore, err = os.Getwd()
	if err != nil {
		t.Error(err)
	}
	os.Chdir(dir)
}

func resetWorkDir() {
	os.Chdir(workDirBefore)
}

func testReset(t *testing.T) {
	resetCmdObj := ResetCmd{}
	resetCmdObj.Run(nil, []string{})

	dir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(path.Join(dir, "Dockerfile"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".dockerignore"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".devspace"))
	assert.Equal(t, true, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, "chart"))
	assert.Equal(t, true, os.IsNotExist(err))

	_, err = os.Stat(path.Join(dir, "index.js"))
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
