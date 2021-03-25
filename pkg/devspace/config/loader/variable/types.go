package variable

import "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

// Variable defines an interface to load a variable
type Variable interface {
	Load(definition *latest.Variable) (interface{}, error)
}

// Resolver defines an interface to resolve defined variables
type Resolver interface {
	// Resolves a single variable by name and possible definition
	Resolve(name string, definition *latest.Variable) (interface{}, error)

	// Convert several variables from input flags in the form of varname=value
	ConvertFlags(flags []string) (map[string]interface{}, error)

	// FindVariables returns all variable names that were found in the given map
	FindVariables(haystack map[interface{}]interface{}, vars []*latest.Variable) (map[string]bool, error)

	// FillVariables walks over the haystack and replaces all encountered variables
	FillVariables(haystack map[interface{}]interface{}) error

	// Replaces all variables in a string and returns either a string, integer or boolean
	ReplaceString(str string) (interface{}, error)

	// Returns the internal memory cache of the resolver with the resolved variables
	ResolvedVariables() map[string]interface{}
}
