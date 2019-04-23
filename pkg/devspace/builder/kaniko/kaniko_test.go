package kaniko

import "testing"

const testNamespace = "test-kaniko-build"

func TestKanikoBuildWithEntrypointOverride(t *testing.T) {
	// @Florian
	// 1. Write test dockerfile and context to a temp folder
	// 2. Create kubectl client
	// 3. Create test namespace test-kaniko-build
	// 4. Build image with kaniko, but don't push it (In kaniko options use "--no-push" as extra flag)
	// 5. Delete temp files & test namespace
}
