package pullsecrets

/*type createPullSecretTestCase struct {
	name string

	namespace       string
	serviceAccounts []string
	imagesInConfig  map[string]*latest.ImageConfig

	expectedLog string
	expectedErr string
}

func TestCreatePullSecrets(t *testing.T) {
	testCases := []createPullSecretTestCase{
		createPullSecretTestCase{
			name:            "One simple creation without default service account",
			namespace:       "testNS",
			serviceAccounts: []string{"someServiceAccount"},
			imagesInConfig: map[string]*latest.ImageConfig{
				"testimage": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            "testimage",
				},
			},
			expectedLog: `
StartWait Creating image pull secret for registry: hub.docker.com
StopWait
Error Couldn't find service account 'default' in namespace 'testNS': serviceaccounts "default" not found`,
		},
		createPullSecretTestCase{
			name:            "One simple creation with default service account",
			namespace:       "testNS",
			serviceAccounts: []string{"default"},
			imagesInConfig: map[string]*latest.ImageConfig{
				"testimage": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            "testimage",
				},
			},
			expectedLog: `
StartWait Creating image pull secret for registry: hub.docker.com
StopWait`,
		},
	}

	for _, testCase := range testCases {
		//Setting up kubeClient
		kubeClient := &kubectl.Client{
			Client:    fake.NewSimpleClientset(),
			Namespace: testCase.namespace,
		}
		_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testCase.namespace,
			},
		})
		assert.NilError(t, err, "Error creating namespace in testCase %s", testCase.name)
		for _, serviceAccount := range testCase.serviceAccounts {
			_, err = kubeClient.Client.CoreV1().ServiceAccounts(testCase.namespace).Create(&k8sv1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: serviceAccount,
				},
			})
			assert.NilError(t, err, "Error creating serviceAccount in testCase %s", testCase.name)
		}

		// Create fake devspace config
		testConfig := &latest.Config{
			Images:      testCase.imagesInConfig,
			Deployments: []*latest.DeploymentConfig{},
		}

		//Unfortunately we can't fake dockerClients yet.
		err = CreatePullSecrets(testConfig, kubeClient, nil, log.Discard)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error creating pull secrets in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error creating pull secrets in testCase %s", testCase.name)
		}
	}

	//TODO: Fake a dockerClient to make this work
	/*secretNames := GetPullSecretNames()
	assert.Equal(t, 1, len(secretNames), "Wrong number of secret names after creating one secret.")
	assert.Equal(t, "devspace-auth-docker", secretNames[0], "Wrong saved sercet name")

	resultSecret , err := kubeClient.CoreV1().Secrets(namespace).Get(secretNames[0], metav1.GetOptions{})
	assert.Equal(t, "devspace-auth-docker", resultSecret.ObjectMeta.Name, "Saved secret has wrong name")
	assert.Equal(t, `{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "` + base64.StdEncoding.EncodeToString([]byte("someuser:password")) + `",
					"email": "someuser@example.com"
				}
			}
		}`, string(resultSecret.Data[k8sv1.DockerConfigJsonKey]), "Saved secret has wrong data")*/ /*

}*/
