package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configs"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	homedir "github.com/mitchellh/go-homedir"
	yaml "gopkg.in/yaml.v2"
)

// ComponentsRepoURL is the repository url
const ComponentsRepoURL = "https://github.com/devspace-cloud/components.git"

// ComponentsRepoPath is the path relative to the user folder where the components are stored
const ComponentsRepoPath = ".devspace/components"

// ComponentsGenerator holds the information to create a component
type ComponentsGenerator struct {
	LocalPath string
	gitRepo   *GitRepository
}

// ComponentSchema is the component schema
type ComponentSchema struct {
	Name           string             `yaml:"name"`
	Description    string             `yaml:"description"`
	Variables      []configs.Variable `yaml:"variables"`
	VariableValues map[string]string
}

// VarMatchRegex is the regex to check if a value matches the devspace var format
var VarMatchRegex = regexp.MustCompile("^(.*)(\\$\\{[^\\}]+\\})(.*)$")

func (c *ComponentSchema) varMatchFn(path, key, value string) bool {
	return VarMatchRegex.MatchString(value)
}

func (c *ComponentSchema) varReplaceFn(path, value string) interface{} {
	matched := VarMatchRegex.FindStringSubmatch(value)
	if len(matched) != 4 {
		return ""
	}

	value = matched[2]
	varName := strings.TrimSpace(value[2 : len(value)-1])
	if _, ok := c.VariableValues[varName]; ok == false {
		// Get variable from component
		variable := &configs.Variable{
			Name:     &varName,
			Question: ptr.String("Please enter a value for " + varName),
		}
		for _, v := range c.Variables {
			if v.Name != nil && *v.Name == varName {
				variable = &v
				break
			}
		}

		// Fill c.VariableValues[varName]
		c.askQuestion(variable)
	}

	retValue := matched[1] + c.VariableValues[varName] + matched[3]

	// Check if we can convert configVal
	if i, err := strconv.Atoi(retValue); err == nil {
		return i
	} else if b, err := strconv.ParseBool(retValue); err == nil {
		return b
	}

	return retValue
}

// askQuestion asks the user a question depending on the variable options
func (c *ComponentSchema) askQuestion(variable *configs.Variable) {
	params := &stdinutil.GetFromStdinParams{}

	if variable == nil {
		params.Question = "Please enter a value"
	} else {
		if variable.Question == nil {
			params.Question = "Please enter a value"
		} else {
			params.Question = *variable.Question
		}

		if variable.Default != nil {
			params.DefaultValue = *variable.Default
		}

		if variable.Options != nil {
			params.Options = *variable.Options
		} else if variable.RegexPattern != nil {
			params.ValidationRegexPattern = *variable.RegexPattern
		}
	}

	c.VariableValues[*variable.Name] = *stdinutil.GetFromStdin(params)
}

// NewComponentGenerator creates a new component generator for the given path
func NewComponentGenerator() (*ComponentsGenerator, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	gitRepository := NewGitRepository(filepath.Join(homedir, ComponentsRepoPath), ComponentsRepoURL)
	_, err = gitRepository.Update()
	if err != nil {
		return nil, err
	}

	return &ComponentsGenerator{
		gitRepo: gitRepository,
	}, nil
}

// ListComponents returns an array with all available components
func (cg *ComponentsGenerator) ListComponents() ([]*ComponentSchema, error) {
	// Check if component exists
	components, err := ioutil.ReadDir(filepath.Join(cg.gitRepo.LocalPath, "components"))
	if err != nil {
		return nil, err
	}

	retArr := make([]*ComponentSchema, 0, len(components))
	for _, component := range components {
		c, err := cg.GetComponent(component.Name())
		if err != nil {
			return nil, err
		}

		retArr = append(retArr, c)
	}

	return retArr, nil
}

// GetComponent retrieves a component
func (cg *ComponentsGenerator) GetComponent(name string) (*ComponentSchema, error) {
	// Check if component exists
	componentFile := filepath.Join(cg.gitRepo.LocalPath, "components", name, "component.yaml")
	_, err := os.Stat(componentFile)
	if err != nil {
		return nil, fmt.Errorf("Component %s does not exist", name)
	}

	// Load component
	yamlFileContent, err := ioutil.ReadFile(componentFile)
	if err != nil {
		return nil, err
	}

	component := &ComponentSchema{}
	err = yaml.UnmarshalStrict(yamlFileContent, component)
	if err != nil {
		return nil, fmt.Errorf("Error loading component: %v", err)
	}

	component.VariableValues = make(map[string]string)
	return component, nil
}

// GetComponentTemplate retrieves a component templates
func (cg *ComponentsGenerator) GetComponentTemplate(name string) (*latest.ComponentConfig, error) {
	component, err := cg.GetComponent(name)
	if err != nil {
		return nil, err
	}

	// Ask questions
	for _, variable := range component.Variables {
		component.askQuestion(&variable)
	}

	// Check if component exists
	componentTemplateFile := filepath.Join(cg.gitRepo.LocalPath, "components", name, "template.yaml")
	_, err = os.Stat(componentTemplateFile)
	if err != nil {
		return nil, fmt.Errorf("Component Template %s does not exist", name)
	}

	// Load component
	yamlFileContent, err := ioutil.ReadFile(componentTemplateFile)
	if err != nil {
		return nil, err
	}

	yamlFileContent, err = configutil.CustomResolveVars(yamlFileContent, component.varMatchFn, component.varReplaceFn)
	if err != nil {
		return nil, fmt.Errorf("Error resolving variables: %v", err)
	}

	componentTemplate := &latest.ComponentConfig{}
	err = yaml.UnmarshalStrict(yamlFileContent, componentTemplate)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling yaml: %v", err)
	}

	return componentTemplate, nil
}
