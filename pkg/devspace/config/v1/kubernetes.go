package v1

type kubernetesInfo struct {
	RegexPatterns *kubernetesRegexPatterns
}

type kubernetesRegexPatterns struct {
	Name string
}

var Kubernetes = &kubernetesInfo{
	RegexPatterns: &kubernetesRegexPatterns{
		Name: "^[a-z][a-z0-9-]{0,50}[a-z0-9]$",
	},
}
