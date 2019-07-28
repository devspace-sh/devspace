package cloud

import (
	"fmt"
	"regexp"
	"strings"
	"encoding/json"
	"encoding/base64"
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
)

var analyticsHostRegex = regexp.MustCompile(`^http(s?)://([^/]+)/?.*$`)

func analyticsIdentify(provider *Provider) error {
	analytics := analytics.GetAnalytics()

	host := analyticsHostRegex.ReplaceAllString(provider.Host, `$2`)

	tokenStringSections := strings.Split(provider.Token, ".")

	encodedToken := tokenStringSections[1]

	if i := len(encodedToken) % 4; i != 0 {
        encodedToken += strings.Repeat("=", 4-i)
    }

	tokenString, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return fmt.Errorf("Unable to decode auth token for analytics: %v", err)
	}
	
	tokenData := map[string]interface{}{}
	if err := json.Unmarshal(tokenString, &tokenData); err != nil {
		return fmt.Errorf("Unable to unmarshal auth token for analytics: %v", err)
	}

	tokenClaims, _ := tokenData["https://hasura.io/jwt/claims"].(map[string]interface{})
	identifier, _ := tokenClaims["x-hasura-user-id"].(string)
	
	return analytics.Identify(host + "/" + identifier)
}