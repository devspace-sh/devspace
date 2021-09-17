package pullsecrets

/*func TestCreatePullSecret(t *testing.T) {
	namespace := "myns"
	//Setting up kubeClient
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	err = CreatePullSecret(kubeClient, namespace, "", "someuser", "password", "someuser@example.com", log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	resultSecret, err := kubeClient.Client.CoreV1().Secrets(namespace).Get("devspace-auth-docker", metav1.GetOptions{})
	assert.Equal(t, "devspace-auth-docker", resultSecret.ObjectMeta.Name, "Saved secret has wrong name")
	assert.Equal(t, `{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "`+base64.StdEncoding.EncodeToString([]byte("someuser:password"))+`",
					"email": "someuser@example.com"
				}
			}
		}`, string(resultSecret.Data[k8sv1.DockerConfigJsonKey]), "Saved secret has wrong data")
}*/
