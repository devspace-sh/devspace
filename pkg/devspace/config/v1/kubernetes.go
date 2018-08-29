package v1

type kubernetesInfo struct {
	RegexPatterns *kubernetesRegexPatterns
}

type kubernetesRegexPatterns struct {
	Name string
}

// Kubernetes is a var that contains all regexes for names given to kubernetes objects
var Kubernetes = &kubernetesInfo{
	RegexPatterns: &kubernetesRegexPatterns{
		Name: "^[a-z][a-z0-9-]{0,50}[a-z0-9]$",
	},
}
