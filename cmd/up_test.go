package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/covexo/devspace/pkg/devspace/cloud"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/helm"
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
/*func TestUpWithInternalRegistry(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseInternalRegistry"), t)
	defer resetWorkDir(t)

	upCmdObj := UpCmd{
		flags: UpFlagsDefault,
	}
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
		t.Error(err)
	}
	t.Logf("read %d bytes: %q\n", count, data[:count])
	assert.Contains(t, string(data[:count]), "3000:3000, 9229:9229", "No js-PortForwarding")
	assert.Contains(t, string(data[:count]), "./           /app", "No Sync")
	assert.Contains(t, string(data[:count]), "No entries found", "A package appeared despite not configured")

	resetCmdObj := ResetCmd{}
	resetCmdObj.kubectl, err = kubectl.NewClient()
	if err != nil {
		t.Error(err)
	}

	resetCmdObj.deleteDevSpaceDeployments()
	resetCmdObj.deleteInternalRegistry()
	resetCmdObj.deleteTiller()
	resetCmdObj.deleteClusterRoleBinding()

	downCmdObj := DownCmd{
		flags: &DownCmdFlags{},
	}
	downCmdObj.Run(nil, nil)

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

	assert.Equal(t, false, helm.IsTillerDeployed(resetCmdObj.kubectl), "Tiller deleted")
	assert.Nil(t, configutil.GetConfig().InternalRegistry, "Internal Registry deleted")
	_, err = resetCmdObj.kubectl.RbacV1beta1().ClusterRoleBindings().Get(kubectl.ClusterRoleBindingName, metav1.GetOptions{})
	assert.Nil(t, configutil.GetConfig().InternalRegistry, "Role Binding in Minikube")

}*/

func TestUpWithDockerHub(t *testing.T) {
	createTempFolderCopy(path.Join(fsutil.GetCurrentGofileDir(), "..", "testData", "cmd", "up", "UseDockerHub"), t)
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
		t.Error(err)
	}
	t.Logf("read %d bytes: %q\n", count, data[:count])
	assert.Contains(t, string(data[:count]), "3000:3000, 9229:9229", "No js-PortForwarding")
	assert.Contains(t, string(data[:count]), "./           /app", "No Sync")
	assert.Contains(t, string(data[:count]), "No entries found", "A package appeared despite not configured")

	config := configutil.GetConfig()
	providerConfig, err := cloud.ParseCloudConfig()
	if err != nil {
		t.Error(err)
	}
	provider, ok := providerConfig[*config.Cluster.CloudProvider]
	assert.True(t, ok, "Cloudprovider not ok")
	err = cloud.DeleteDevSpace(provider, *config.Cluster.Namespace)
	if err != nil {
		t.Error(err)
	}

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

	kubectlClient, err := kubectl.NewClient()
	if err != nil {
		t.Error(err)
	}

	assert.True(t, helm.IsTillerDeployed(kubectlClient), "Tiller deleted")
	assert.Nil(t, configutil.GetConfig().InternalRegistry, "Internal Registry deleted")
	_, err = kubectlClient.RbacV1beta1().ClusterRoleBindings().Get(kubectl.ClusterRoleBindingName, metav1.GetOptions{})
	assert.Nil(t, configutil.GetConfig().InternalRegistry, "Role Binding in Minikube")

}

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
