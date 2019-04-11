package token

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ClaimSet is the auth token claim set type
type ClaimSet struct {
	Subject  string `json:"sub"`
	Admin    bool   `json:"admin"`
	IssuedAt int64  `json:"iat"`
	Hasura   Hasura `json:"https://hasura.io/jwt/claims"`
}

// Hasura holds the hasura configuration
type Hasura struct {
	AccountID    string   `json:"x-hasura-user-id"`
	DefaultRole  string   `json:"x-hasura-default-role"`
	AllowedRoles []string `json:"x-hasura-allowed-roles"`
}

// Token describes a JSON Web Token.
type Token struct {
	Raw       string
	Claims    *ClaimSet
	Signature []byte
}

// GetAccountID retrieves the account id for the current user from the token
func GetAccountID(token string) (int, error) {
	t, err := ParseTokenClaims(token)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(t.Claims.Hasura.AccountID)
}

// GetAccountName retrieves the account name for the current user
func GetAccountName(token string) (string, error) {
	t, err := ParseTokenClaims(token)
	if err != nil {
		return "", err
	}

	return t.Claims.Subject, nil
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
