package examples

// RunQuickstart runs the test for the quickstart example
func RunQuickstart(f *customFactory) error {
	f.GetLog().Info("Run Quickstart")

	err := RunTest(f, "quickstart", nil)
	if err != nil {
		return err
	}

	return nil
}
