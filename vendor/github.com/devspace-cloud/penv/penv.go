package penv

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// Environment is a collection of appenders, setters and unsetters
	Environment struct {
		Appenders []NameValue
		Setters   []NameValue
		Unsetters []NameValue
	}
	// NameValue is a name value pair
	NameValue struct {
		Name  string
		Value string
	}
)

func uniquei(arr []string) []string {
	u := make([]string, 0, len(arr))
	h := map[string]struct{}{}
	for _, str := range arr {
		stri := strings.ToLower(str)
		if _, ok := h[stri]; ok {
			continue
		}
		h[stri] = struct{}{}
		u = append(u, str)
	}
	return u
}

func filter(arr []NameValue, cond func(NameValue) bool) []NameValue {
	nvs := make([]NameValue, 0, len(arr))
	for _, nv := range arr {
		if cond(nv) {
			nvs = append(nvs, nv)
		}
	}
	return nvs
}

type registeredDAO struct {
	DAO       DAO
	Condition func() bool
	Priority  int
}

type registeredDAOs []registeredDAO

var registered = make(registeredDAOs, 0)

func (r registeredDAOs) Len() int {
	return len(r)
}
func (r registeredDAOs) Less(i, j int) bool {
	return r[i].Priority < r[j].Priority
}
func (r registeredDAOs) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// RegisterDAO registers a new data access object. DAOs are evaluted based on
// priority and the condition function
func RegisterDAO(priority int, condition func() bool, dao DAO) {
	registered = append(registered, registeredDAO{dao, condition, priority})
	sort.Sort(registered)
}

// DAO defines the interface for loading and saving a set of environment
// variables
type DAO interface {
	Load() (*Environment, error)
	Save(*Environment) error
}

// Load loads the environment
func Load() (*Environment, error) {
	for _, r := range registered {
		if r.Condition() {
			return r.DAO.Load()
		}
	}
	return nil, fmt.Errorf("no environment DAO found")
}

// Save saves the environment
func Save(env *Environment) error {
	for _, r := range registered {
		if r.Condition() {
			return r.DAO.Save(env)
		}
	}
	return fmt.Errorf("no environment DAO found")
}

// AppendEnv permanently appends an environment variable
func AppendEnv(name, value string) error {
	env, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load environment: %v", err)
	}
	env.Setters = filter(env.Setters, func(nv NameValue) bool {
		return nv.Name != name || nv.Value != value
	})
	env.Appenders = filter(env.Appenders, func(nv NameValue) bool {
		return nv.Name != name || nv.Value != value
	})
	env.Appenders = append(env.Appenders, NameValue{name, value})
	// if it's being unset, remove it from the list
	env.Unsetters = filter(env.Unsetters, func(nv NameValue) bool {
		return nv.Name != name
	})
	err = Save(env)
	if err != nil {
		return fmt.Errorf("failed to save environment: %v", err)
	}
	return nil
}

// SetEnv permanently sets an environment variable
func SetEnv(name, value string) error {
	env, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load environment: %v", err)
	}
	env.Setters = filter(env.Setters, func(nv NameValue) bool {
		return nv.Name != name
	})
	env.Setters = append(env.Setters, NameValue{name, value})
	env.Unsetters = filter(env.Unsetters, func(nv NameValue) bool {
		return nv.Name != name
	})
	err = Save(env)
	if err != nil {
		return fmt.Errorf("failed to save environment: %v", err)
	}
	return nil
}

// UnsetEnv permanently unsets an environment variable
func UnsetEnv(name string) error {
	env, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load environment: %v", err)
	}
	env.Setters = filter(env.Setters, func(nv NameValue) bool {
		return nv.Name != name
	})
	env.Appenders = filter(env.Appenders, func(nv NameValue) bool {
		return nv.Name != name
	})
	env.Unsetters = filter(env.Unsetters, func(nv NameValue) bool {
		return nv.Name != name
	})
	env.Unsetters = append(env.Unsetters, NameValue{name, ""})

	err = Save(env)
	if err != nil {
		return fmt.Errorf("failed to save environment: %v", err)
	}
	return nil
}
