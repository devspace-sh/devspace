package v1

// Variable describes the var definition
type Variable struct {
	Name         *string `yaml:"name"`
	Default      *string `yaml:"default,omitempty"`
	Question     *string `yaml:"question,omitempty"`
	RegexPattern *string `yaml:"regexPattern,omitempty"`
}
