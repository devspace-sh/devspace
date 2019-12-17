package deploy

import (
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

//Test 3 - kubectl
//1. deploy & kubectl (see quickstart-kubectl)
//2. purge (check if everything is deleted except namespace)

// RunKubectl runs the test for the kubectl test
func RunKubectl(f *customFactory) error {
	log.GetInstance().Info("Run Kubectl")

	ts := testSuite{
		test{
			name: "1. deploy & kubectl (see quickstart-kubectl)",
			deployConfig: &cmd.DeployCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.namespace,
					NoWarn:    true,
				},
			},
			postCheck: nil,
		},
	}

	client, err := f.NewKubeClientFromContext("", f.namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// At last, we delete the current namespace
	defer utils.DeleteNamespaceAndWait(client, f.namespace)

	testDir := filepath.FromSlash("tests/deploy/testdata/kubectl")

	dirPath, _, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	defer utils.DeleteTempAndResetWorkingDir(dirPath, f.pwd)

	// Copy the testdata into the temp dir
	err = utils.Copy(testDir, dirPath)
	if err != nil {
		return err
	}

	// Change working directory
	err = utils.ChangeWorkingDir(dirPath)
	if err != nil {
		return err
	}

	for _, t := range ts {
		err := runTest(f, &t)
		utils.PrintTestResult("kubectl", t.name, err)
		if err != nil {
			return err
		}
	}

	err = testPurge(f)
	utils.PrintTestResult("kubectl", "purge", err)
	if err != nil {
		return err
	}

	return nil
}
