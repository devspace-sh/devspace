package v1

// Configs is the map that specifies all configs
type Configs map[string]*ConfigDefinition

// ConfigDefinition holds the information about a certain config
type ConfigDefinition struct {
	Config     *ConfigWrapper    `yaml:"config,omitempty"`
	Overwrites *[]*ConfigWrapper `yaml:"overwrites,omitempty"`
}

// ConfigWrapper specifies if the config is infile or should be loaded from a path
type ConfigWrapper struct {
	Path *string `yaml:"path,omitempty"`
	Data *Config `yaml:"data,omitempty"`
}
