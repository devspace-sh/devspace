package patch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/yamlutil"

	"gopkg.in/yaml.v3"
)

type Patch []Operation

// Apply returns a YAML document that has been mutated per patch
func (p Patch) Apply(doc []byte) ([]byte, error) {
	var node yaml.Node
	err := yamlutil.Unmarshal(doc, &node)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%s", string(doc), err)
	}

	for _, op := range p {
		err = op.Perform(&node)
		if err != nil {
			return nil, err
		}
	}

	return yaml.Marshal(&node)
}

func NewNode(raw *interface{}) (*yaml.Node, error) {
	doc, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling struct: %+v\n\n%s", raw, err)
	}

	var node yaml.Node
	err = yamlutil.Unmarshal(doc, &node)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling doc: %s\n\n%s", string(doc), err)
	}

	return &node, nil
}

func TransformPath(path string) string {
	if path == "" {
		return path
	}

	legacyExtendedSyntaxRegEx := regexp.MustCompile(`(?i)([^\=]+)=([^\.\=\>\<\~]+)`)
	hasFilterRegEx := regexp.MustCompile(`(?i)\[\?.*\)\]`)
	indexXPathRegEx := regexp.MustCompile(`\/(\d+|\*)\/`)
	trailingIndexXPathRegEx := regexp.MustCompile(`\/(\d+|\*)$`)
	rootXPathRegEx := regexp.MustCompile(`^\/`)
	numeric := regexp.MustCompile(`^\d+$`)
	rewrittenPath := path

	if legacyExtendedSyntaxRegEx.MatchString(path) {
		// Using property=value selectors
		rewriteTokens := []string{}
		tokens := strings.Split(path, ".")
		for _, token := range tokens {
			rewriteToken := token
			if legacyExtendedSyntaxRegEx.MatchString(token) {
				filterTokens := legacyExtendedSyntaxRegEx.FindStringSubmatch(token)
				if numeric.MatchString((filterTokens[2])) {
					rewriteToken = fmt.Sprintf("[?(@.%s=='%s' || @.%s==%s)]", filterTokens[1], filterTokens[2], filterTokens[1], filterTokens[2])
				} else {
					rewriteToken = fmt.Sprintf("[?(@.%s=='%s')]", filterTokens[1], filterTokens[2])
				}
			}
			rewriteTokens = append(rewriteTokens, rewriteToken)
		}
		rewrittenPath = strings.Join(rewriteTokens, ".")
		rewrittenPath = strings.ReplaceAll(rewrittenPath, ".[?", "[?")
	} else if strings.Contains(path, "/") && !hasFilterRegEx.MatchString(path) {
		// Is XPath
		rewrittenPath = indexXPathRegEx.ReplaceAllString(path, "[$1].")
		rewrittenPath = trailingIndexXPathRegEx.ReplaceAllString(rewrittenPath, "[$1]")
		rewrittenPath = rootXPathRegEx.ReplaceAllLiteralString(rewrittenPath, "$.")
	}

	return rewrittenPath
}
