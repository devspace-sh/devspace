package examples

// RunKustomize runs the test for the kustomize example
func RunKustomize(f *customFactory) error {
	f.GetLog().Info("Run Kustomize")

	err := RunTest(f, "kustomize", nil)
	if err != nil {
		return err
	}

	return nil
}
