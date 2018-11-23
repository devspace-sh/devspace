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

/*The internal registry is not supported in the init command.
However, it is still supported if written in the config.yaml .
And it should work, which is why it is tested here. */
func TestUpWithInternalRegistry(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseInternalRegistry"), t)
	defer resetWorkDir(t)

	upCmdObj := UpCmd{
		flags: UpFlagsDefault,
	}
	upCmdObj.flags.sync = false
	upCmdObj.flags.exitAfterDeploy = true

	defer func() {
		client, err := kubectl.NewClient()
		if err != nil {
			t.Error(err)
		}
		propagationPolicy := metav1.DeletePropagationForeground
		client.Core().Namespaces().Delete("test-cmd-up-private-registry", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}()

	upCmdObj.Run(nil, []string{})

	log.StartFileLogging()
	listCmdObj := ListCmd{
		flags: &ListCmdFlags{},
	}
	listCmdObj.RunListPort(nil, nil)
	listCmdObj.RunListService(nil, nil)
	listCmdObj.RunListSync(nil, nil)
	listCmdObj.RunListPackage(nil, nil)

	logFile, err := os.Open(log.Logdir + "default.log")
	if err != nil {
		t.Error(err)
		return
	}
	data := make([]byte, 10000)
	count, err := logFile.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	t.Logf("read %d bytes: %q\n", count, data[:count])
	assert.Contains(t, string(data[:count]), "pod    release=devspace-test-cmd-up-private-registry   3000:3000", "No PortForwarding")
	assert.Contains(t, string(data[:count]), "No services are configured. Run `devspace add service` to add new service", "A service appeared despite not configured")
	assert.Contains(t, string(data[:count]), "release=devspace-test-cmd-up-private-registry   ./           /app", "No Sync")
	assert.Contains(t, string(data[:count]), "No entries found", "A package appeared despite not configured")

	resetCmdObj := ResetCmd{
		flags: &ResetCmdFlags{
			skipQuestionsWithGivenAnswers: true,
			deleteFromDevSpaceCloud:       false,
			removeCloudContext:            false,
			removeTiller:                  false,
			deleteChart:                   false,
			removeRegistry:                false,
			deleteDockerfile:              false,
			deleteDockerIgnore:            false,
			deleteRoleBinding:             false,
			deleteDevspaceFolder:          false,
		},
	}
	resetCmdObj.Run(nil, []string{})

	dir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(path.Join(dir, "Dockerfile"))
	assert.Equal(t, false, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".dockerignore"))
	assert.Equal(t, false, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, ".devspace"))
	assert.Equal(t, false, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, "chart"))
	assert.Equal(t, false, os.IsNotExist(err))

	_, err = os.Stat(path.Join(dir, "index.js"))
	assert.Equal(t, false, os.IsNotExist(err))
	_, err = os.Stat(path.Join(dir, "package.json"))
	assert.Equal(t, false, os.IsNotExist(err))

}

/*func TestUpWithDockerHub(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseDockerHub"), t)
	defer resetWorkDir(t)

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
		client.Core().Namespaces().Delete("217b737767c3420e68e6c3b659eb46bb", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	}()

	upCmdObj.Run(nil, []string{})
	log.StopFileLogging()

	testReset(t)

}*/

var workDirBefore string

func createTempFolderCopy(source string, t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	err = fsutil.Copy(source, dir, true)
	if err != nil {
		t.Error(err)
	}

	workDirBefore, err = os.Getwd()
	if err != nil {
		t.Error(err)
	}
	os.Chdir(dir)
}

func resetWorkDir(t *testing.T) {
	tmpDir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	os.Chdir(workDirBefore)

	os.Remove(tmpDir)
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
