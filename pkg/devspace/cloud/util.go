package cloud

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"

	"github.com/devspace-cloud/devspace/pkg/util/envutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// PrintSpaces prints the users spaces
func (p *Provider) PrintSpaces(name string) error {
	spaces, err := p.GetSpaces()
	if err != nil {
		return fmt.Errorf("Error retrieving spaces: %v", err)
	}

	activeSpaceID := 0
	if configutil.ConfigExists() {
		generated, err := generated.LoadConfig()
		if err == nil && generated.CloudSpace != nil {
			activeSpaceID = generated.CloudSpace.SpaceID
		}
	}

	headerColumnNames := []string{}
	if activeSpaceID != 0 {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Active",
			"Domain",
			"Created",
		}...)
	} else {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Domain",
			"Created",
		}...)
	}

	values := [][]string{}

	for _, space := range spaces {
		if name == "" || name == space.Name {
			created, err := time.Parse(time.RFC3339, strings.Split(space.Created, ".")[0]+"Z")
			if err != nil {
				return err
			}

			domain := ""
			if space.Domain != nil {
				domain = *space.Domain
			}

			if activeSpaceID != 0 {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					strconv.FormatBool(space.SpaceID == activeSpaceID),
					domain,
					created.String(),
				})
			} else {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					domain,
					created.String(),
				})
			}
		}
	}

	if len(values) > 0 {
		log.PrintTable(headerColumnNames, values)
	} else {
		log.Info("No spaces found")
	}

	return nil
}

// SetTillerNamespace sets the tiller environment variable
func SetTillerNamespace(space *Space) error {
	if space == nil {
		return envutil.SetEnvVar("TILLER_NAMESPACE", "kube-system")
	}
	return envutil.SetEnvVar("TILLER_NAMESPACE", space.Namespace)
}

// ParseTokenClaims parses a token from a string
func ParseTokenClaims(rawToken string) (*Token, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Token is malformed, expected 3 parts got %d", len(parts))
	}

	var (
		rawClaims  = parts[1]
		claimsJSON []byte
		err        error
	)

	if claimsJSON, err = joseBase64UrlDecode(rawClaims); err != nil {
		return nil, fmt.Errorf("unable to decode claims: %s", err)
	}

	retToken := new(Token)
	retToken.Claims = new(ClaimSet)

	retToken.Raw = strings.Join(parts[:2], ".")
	if retToken.Signature, err = joseBase64UrlDecode(parts[2]); err != nil {
		return nil, fmt.Errorf("unable to decode signature: %s", err)
	}

	if err = json.Unmarshal(claimsJSON, retToken.Claims); err != nil {
		return nil, fmt.Errorf("unable to unmarshal claims: %s", err)
	}

	return retToken, nil
}

// joseBase64UrlDecode decodes the given string using the standard base64 url
// decoder but first adds the appropriate number of trailing '=' characters in
// accordance with the jose specification.
// http://tools.ietf.org/html/draft-ietf-jose-json-web-signature-31#section-2
func joseBase64UrlDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 0:
	case 2:
		s += "=="
	case 3:
		s += "="
	default:
		return nil, errors.New("illegal base64url string")
	}
	return base64.URLEncoding.DecodeString(s)
}
