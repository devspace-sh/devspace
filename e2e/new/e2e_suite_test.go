package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"math/rand"
	"testing"
	"time"

	// Register tests
	_ "github.com/loft-sh/devspace/e2e/new/tests/build"
	_ "github.com/loft-sh/devspace/e2e/new/tests/config"
	_ "github.com/loft-sh/devspace/e2e/new/tests/dependencies"
	_ "github.com/loft-sh/devspace/e2e/new/tests/deploy"
	_ "github.com/loft-sh/devspace/e2e/new/tests/sync"
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
