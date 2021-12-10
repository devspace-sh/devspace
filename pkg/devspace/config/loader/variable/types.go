package variable

import "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

// Variable defines an interface to load a variable
type Variable interface {
	Load(definition *latest.Variable) (interface{}, error)
}

// Resolver defines an interface to resolve defined variables
type Resolver interface {
	// ConvertFlags converts several variables from input flags in the form of varname=value
	ConvertFlags(flags []string) (map[string]interface{}, error)

	// FindVariables returns all variable names that were found in the given map
	FindVariables(haystack interface{}, vars []*latest.Variable) (map[string]bool, error)

	// FillVariables finds the used variables first and then fills in those in the haystack
	FillVariables(haystack interface{}, vars []*latest.Variable) (interface{}, error)

	// ResolvedVariables returns the internal memory cache of the resolver with all resolved variables
	ResolvedVariables() map[string]interface{}
}
