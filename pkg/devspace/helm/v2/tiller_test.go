package v2

import ()

/*func createTestResources(client kubernetes.Interface) error {
	podMetadata := metav1.ObjectMeta{
		Name: "test-pod",
		Labels: map[string]string{
			"app.kubernetes.io/name": "devspace-app",
		},
	}
	podSpec := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "test",
				Image: "nginx",
			},
		},
	}

	deploy := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: TillerDeploymentName},
		Spec: v1beta1.DeploymentSpec{
			Replicas: ptr.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "devspace-app",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: podMetadata,
				Spec:       podSpec,
			},
		},
		Status: v1beta1.DeploymentStatus{
			AvailableReplicas:  1,
			ObservedGeneration: 1,
			ReadyReplicas:      1,
			Replicas:           1,
			UpdatedReplicas:    1,
		},
	}
	_, err := client.AppsV1().Deployments(loader.TestNamespace).Create(deploy)
	if err != nil {
		return errors.Wrap(err, "create deployment")
	}

	return nil
}

func TestTillerEnsure(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// Inject an event into the fake client.
	err := createTestResources(client.KubeClient())
	if err != nil {
		t.Fatal(err)
	}

	err = ensureTiller(config, client, loader.TestNamespace, true, log.Discard)
	if err != nil {
		t.Fatal(err)
	}

	isTillerDeployed := IsTillerDeployed(config, client, loader.TestNamespace)
	if isTillerDeployed == false {
		t.Fatal("Expected that tiller is deployed")
	}

	//Break deployment
	deployment, err := client.KubeClient().AppsV1().Deployments(loader.TestNamespace).Get(TillerDeploymentName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error breaking deployment: %v", err)
	}
	deployment.Status.Replicas = 1
	deployment.Status.ReadyReplicas = 2
	client.KubeClient().AppsV1().Deployments(loader.TestNamespace).Update(deployment)

	isTillerDeployed = IsTillerDeployed(config, client, loader.TestNamespace)
	assert.Equal(t, false, isTillerDeployed, "Tiller declared deployed despite deployment being broken")
}

func TestTillerCreate(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	tillerOptions := getTillerOptions(loader.TestNamespace)

	err := createTiller(config, client, loader.TestNamespace, tillerOptions, log.Discard)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTillerDelete(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// Inject an event into the fake client.
	err := DeleteTiller(config, client, loader.TestNamespace)
	if err != nil {
		t.Fatal(err)
	}
}*/
