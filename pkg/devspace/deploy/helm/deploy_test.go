package helm

import "testing"

// Test namespace to create
const testNamespace = "test-helm-deploy"

func TestHelmDeployment(t *testing.T) {
	// @Florian
	// 1. Create fake config & generated config
	// 2. Write test chart into a temp folder
	// 3. Init kubectl & create test namespace
	// 4. Deploy test chart
	// 5. Validate deployed chart & test .Status function
	// 6. Delete test chart
	// 7. Delete test namespace
	// 8. Delete temp folder
}
