package examples

// RunMicroservices runs the test for the kustomize example
func RunMicroservices(f *customFactory) error {
	f.GetLog().Info("Run Microservices")

	err := RunTest(f, "microservices", nil)
	if err != nil {
		return err
	}

	return nil
}
