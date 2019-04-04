package configs

// Configs is the map that specifies all configs
type Configs map[string]*ConfigDefinition

// ConfigDefinition holds the information about a certain config
type ConfigDefinition struct {
	Config    *ConfigWrapper    `yaml:"config,omitempty"`
	Vars      *VarsWrapper      `yaml:"vars,omitempty"`
	Overrides *[]*ConfigWrapper `yaml:"overrides,omitempty"`
}

// ConfigWrapper specifies if the config is infile or should be loaded from a path
type ConfigWrapper struct {
	Path *string     `yaml:"path,omitempty"`
	Data interface{} `yaml:"data,omitempty"`
}

// VarsWrapper specifies if the vars definition is infile or should be loaded from a path
type VarsWrapper struct {
	Path *string      `yaml:"path,omitempty"`
	Data *[]*Variable `yaml:"data,omitempty"`
}

// Variable describes the var definition
type Variable struct {
	Name         *string   `yaml:"name"`
	Options      *[]string `yaml:"options,omitempty"`
	Default      *string   `yaml:"default,omitempty"`
	Question     *string   `yaml:"question,omitempty"`
	RegexPattern *string   `yaml:"regexPattern,omitempty"`
}
