package examples

// RunPhpMysql runs the test for the quickstart example
func RunPhpMysql(f *customFactory) error {
	f.GetLog().Info("Run Php Mysql")

	err := RunTest(f, "php-mysql-example", nil)
	if err != nil {
		return err
	}

	return nil
}
