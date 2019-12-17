package examples

// RunMinikube runs the test for the kustomize example
func RunMinikube(f *customFactory) error {
	f.GetLog().Info("Run Minikube")

	err := RunTest(f, "minikube", nil)
	if err != nil {
		return err
	}

	return nil
}
