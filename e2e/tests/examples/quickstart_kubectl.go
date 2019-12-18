package examples

// RunQuickstartKubectl runs the test for the quickstart example
func RunQuickstartKubectl(f *customFactory) error {
	f.GetLog().Info("Run Quickstart Kubectl")

	err := RunTest(f, "quickstart-kubectl", nil)
	if err != nil {
		return err
	}

	return nil
}
