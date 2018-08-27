package envutil

func SetEnvVar(name string, value string) error {
	return setEnv(name, value)
}
