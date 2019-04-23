package kubectl

import "testing"

// Test namespace to create
const testNamespace = "test-kubectl-deploy"

// Test namespace to create
const testKustomizeNamespace = "test-kubectl-kustomize-deploy"

func TestKubectlManifests(t *testing.T) {
	// @Florian
	// 1. Create fake config & generated config
	// 2. Write test manifests into a temp folder
	// 3. Init kubectl & create test namespace
	// 4. Deploy manifests
	// 5. Validate manifests
	// 6. Delete manifests
	// 7. Delete test namespace
	// 8. Delete temp folder
}

func TestKubectlManifestsWithKustomize(t *testing.T) {
	// @Florian
	// 1. Create fake config & generated config
	// 2. Write test kustomize files (see examples) into a temp folder
	// 3. Init kubectl & create test namespace
	// 4. Deploy files
	// 5. Validate deployed resources
	// 6. Delete deployed files
	// 7. Delete test namespace
	// 8. Delete temp folder
}
