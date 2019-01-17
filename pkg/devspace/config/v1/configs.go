package v1

// Configs is the map that specifies all configs
type Configs map[string]*ConfigDefinition

// ConfigDefinition holds the information about a certain config
type ConfigDefinition struct {
	Config     *ConfigWrapper    `yaml:"config,omitempty"`
	Vars       *VarsWrapper      `yaml:"vars,omitempty"`
	Overwrites *[]*ConfigWrapper `yaml:"overwrites,omitempty"`
}

// ConfigWrapper specifies if the config is infile or should be loaded from a path
type ConfigWrapper struct {
	Path *string                     `yaml:"path,omitempty"`
	Data map[interface{}]interface{} `yaml:"data,omitempty"`
}

// VarsWrapper specifies if the vars definition is infile or should be loaded from a path
type VarsWrapper struct {
	Path *string      `yaml:"path,omitempty"`
	Data *[]*Variable `yaml:"data,omitempty"`
}
