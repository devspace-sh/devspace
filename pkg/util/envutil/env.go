package envutil

//SetEnvVar sets an environment variable
func SetEnvVar(name string, value string) error {
	return setEnv(name, value)
}
