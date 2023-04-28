package patch

import (
	"testing"
)

type applyTestCase struct {
	input  string
	output string
	patch  Patch
}

func TestApply(t *testing.T) {
	testCases := map[string]applyTestCase{
		// If it were processed, the indentation would be reset to 4 spaces
		"noop no patches": {
			input: `
apiVersion: v1
kind: Pod
metadata:
  name: vela-core-application-test
  namespace: vela-system
spec:
  containers:
  - command:
    - /bin/bash
    - -ec
    - |2

      set -e

      echo "Application and its components are created"
    image: oamdev/alpine-k8s:1.18.2
    imagePullPolicy: IfNotPresent
    name: vela-core-application-test
  restartPolicy: Never
  serviceAccountName: vela-core
`,
			output: `
apiVersion: v1
kind: Pod
metadata:
  name: vela-core-application-test
  namespace: vela-system
spec:
  containers:
  - command:
    - /bin/bash
    - -ec
    - |2

      set -e

      echo "Application and its components are created"
    image: oamdev/alpine-k8s:1.18.2
    imagePullPolicy: IfNotPresent
    name: vela-core-application-test
  restartPolicy: Never
  serviceAccountName: vela-core
`,
		},
		// TODO Failing test cases
		//		"applies patch": {
		//			input: `apiVersion: v1
		//kind: Pod
		//metadata:
		//  annotations:
		//    helm.sh/hook: test
		//    helm.sh/hook-delete-policy: hook-succeeded
		//  name: vela-core-application-test
		//  namespace: vela-system
		//spec:
		//  containers:
		//  - command:
		//    - /bin/bash
		//    - -ec
		//    - |2
		//
		//      set -e
		//
		//      echo "Application and its components are created"
		//    image: oamdev/alpine-k8s:1.18.2
		//    imagePullPolicy: IfNotPresent
		//    name: vela-core-application-test
		//  restartPolicy: Never
		//  serviceAccountName: vela-core
		//`,
		//			output: `apiVersion: v1
		//kind: Pod
		//metadata:
		//  annotations:
		//    helm.sh/hook: test
		//    helm.sh/hook-delete-policy: hook-succeeded
		//  name: vela-core-application-test
		//  namespace: vela-system
		//spec:
		//  containers:
		//    - command:
		//        - /bin/bash
		//        - -ec
		//        - |2
		//          set -e
		//
		//          echo "Application and its components are created"
		//      image: oamdev/alpine-k8s:1.18.2
		//      imagePullPolicy: IfNotPresent
		//      name: vela-core-application-test
		//  serviceAccountName: vela-core
		//`,
		//			patch: []Operation{
		//				{
		//					Op:   opRemove,
		//					Path: "spec.restartPolicy",
		//				},
		//			},
		//		},
		//		"applies patch 4 spaces": {
		//			input: `apiVersion: v1
		//kind: Pod
		//metadata:
		//    annotations:
		//        helm.sh/hook: test
		//        helm.sh/hook-delete-policy: hook-succeeded
		//    name: vela-core-application-test
		//    namespace: vela-system
		//spec:
		//    containers:
		//        - command:
		//              - /bin/bash
		//              - -ec
		//              - |2
		//
		//                set -e
		//
		//                echo "Application and its components are created"
		//          image: oamdev/alpine-k8s:1.18.2
		//          imagePullPolicy: IfNotPresent
		//          name: vela-core-application-test
		//    restartPolicy: Never
		//    serviceAccountName: vela-core
		//`,
		//			output: `apiVersion: v1
		//kind: Pod
		//metadata:
		//    annotations:
		//        helm.sh/hook: test
		//        helm.sh/hook-delete-policy: hook-succeeded
		//    name: vela-core-application-test
		//    namespace: vela-system
		//spec:
		//    containers:
		//        - command:
		//              - /bin/bash
		//              - -ec
		//              - |2
		//                set -e
		//
		//                echo "Application and its components are created"
		//          image: oamdev/alpine-k8s:1.18.2
		//          imagePullPolicy: IfNotPresent
		//          name: vela-core-application-test
		//    serviceAccountName: vela-core
		//`,
		//			patch: []Operation{
		//				{
		//					Op:   opRemove,
		//					Path: "spec.restartPolicy",
		//				},
		//			},
		//		},
	}

	for name, testCase := range testCases {
		output, err := testCase.patch.Apply([]byte(testCase.input))
		if err != nil {
			t.Errorf("Error %v in case %s", err, name)
		}
		if testCase.output != string(output) {
			t.Errorf("TestCase %s\nACTUAL:\n|%s|\nEXPECTED:\n|%s|", name, output, testCase.output)
		}
	}
}
