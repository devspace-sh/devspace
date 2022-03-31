package types

type ProxyCommand struct {
	TTY    bool `json:"tty,omitempty"`
	Height int  `json:"height,omitempty"`
	Width  int  `json:"width,omitempty"`

	Env        []string `json:"env,omitempty"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"workingDir,omitempty"`
}
