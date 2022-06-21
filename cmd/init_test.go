package cmd

import (
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

type parseImagesTestCase struct {
	name      string
	manifests string
	expected  []string
}

func TestParseImages(t *testing.T) {
	testCases := []parseImagesTestCase{
		{
			name: `Single`,
			manifests: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "new"
  labels:
    "app.kubernetes.io/name": "devspace-app"
    "app.kubernetes.io/component": "test"
    "app.kubernetes.io/managed-by": "Helm"
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      "app.kubernetes.io/name": "devspace-app"
      "app.kubernetes.io/component": "test"
      "app.kubernetes.io/managed-by": "Helm"
  template:
    metadata:
      labels:
        "app.kubernetes.io/name": "devspace-app"
        "app.kubernetes.io/component": "test"
        "app.kubernetes.io/managed-by": "Helm"
    spec:
      containers:
        - image: "username/app"
          name: "container-0"
`,
			expected: []string{
				"username/app",
			},
		},
		{
			name: `Multiple`,
			manifests: `
---
# Source: my-app/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: php
  labels:
    release: "test-helm"
spec:
  ports:
  - port: 80
    protocol: TCP
  selector:
    release: "test-helm"
---
# Source: my-app/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-helm
  labels:
    release: "test-helm"
spec:
  replicas: 1
  selector:
    matchLabels:
      release: "test-helm"
  template:
    metadata:
      annotations:
        revision: "1"
      labels:
        release: "test-helm"
    spec:
      containers:
      - name: default
        image: "php"
`,
			expected: []string{
				"php",
			},
		},
	}

	for _, testCase := range testCases {
		manifests := testCase.manifests

		actual, err := parseImages(manifests)
		assert.NilError(
			t,
			err,
			"Unexpected error in test case %s",
			testCase.name,
		)

		expected := testCase.expected
		assert.Assert(
			t,
			cmp.DeepEqual(expected, actual),
			"Unexpected values in test case %s",
			testCase.name,
		)
	}
}
