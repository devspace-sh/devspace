package e2e

import (
	"github.com/onsi/ginkgo/v2"
	"math/rand"
	"testing"
	"time"

	"github.com/onsi/gomega"

	// Register tests
	_ "github.com/loft-sh/devspace/e2e/tests/build"
	_ "github.com/loft-sh/devspace/e2e/tests/command"
	_ "github.com/loft-sh/devspace/e2e/tests/config"
	_ "github.com/loft-sh/devspace/e2e/tests/dependencies"
	_ "github.com/loft-sh/devspace/e2e/tests/deploy"
	_ "github.com/loft-sh/devspace/e2e/tests/devspacehelper"
	_ "github.com/loft-sh/devspace/e2e/tests/hooks"
	_ "github.com/loft-sh/devspace/e2e/tests/imports"
	_ "github.com/loft-sh/devspace/e2e/tests/init"
	_ "github.com/loft-sh/devspace/e2e/tests/localregistry"
	_ "github.com/loft-sh/devspace/e2e/tests/pipelines"
	_ "github.com/loft-sh/devspace/e2e/tests/portforward"
	_ "github.com/loft-sh/devspace/e2e/tests/proxycommands"
	_ "github.com/loft-sh/devspace/e2e/tests/pullsecret"
	_ "github.com/loft-sh/devspace/e2e/tests/render"
	_ "github.com/loft-sh/devspace/e2e/tests/replacepods"
	_ "github.com/loft-sh/devspace/e2e/tests/restarthelper"
	_ "github.com/loft-sh/devspace/e2e/tests/ssh"
	_ "github.com/loft-sh/devspace/e2e/tests/sync"
	_ "github.com/loft-sh/devspace/e2e/tests/terminal"
)

// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func TestRunE2ETests(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "DevSpace e2e suite")
}
