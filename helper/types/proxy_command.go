package types

type ProxyCommand struct {
	Env        []string `json:"env,omitempty"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"workingDir,omitempty"`
}
