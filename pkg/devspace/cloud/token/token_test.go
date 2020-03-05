package token

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestIsTokenValid(t *testing.T) {
	assert.Equal(t, false, IsTokenValid(""), "Empty token is declared as valid")
	assert.Equal(t, false, IsTokenValid(".."), "Token with three empty parts is declared as valid")
	assert.Equal(t, false, IsTokenValid(".a."), "Token with undecodable rawClaims is declared as valid")
	assert.Equal(t, false, IsTokenValid("..a"), "Token with undecodable signature is declared as valid")

	testClaim := ClaimSet{
		Expiration: time.Now().Add(-time.Minute).Unix(),
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(encodedClaim), "=") {
		encodedClaim = strings.TrimSuffix(encodedClaim, "=")
	}

	assert.Equal(t, false, IsTokenValid("."+encodedClaim+"."), "Expired token is declared as valid")

	testClaim = ClaimSet{
		Expiration: time.Now().Add(time.Hour).Unix(),
	}
	claimAsJSON, _ = json.Marshal(testClaim)
	encodedClaim = base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(encodedClaim), "=") {
		encodedClaim = strings.TrimSuffix(encodedClaim, "=")
	}

	assert.Equal(t, true, IsTokenValid("."+encodedClaim+"."), "Valid token is declared as invalid")
}

func TestGetAccountID(t *testing.T) {
	testClaim := ClaimSet{
		Hasura: Hasura{
			AccountID: "1",
		},
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(encodedClaim), "=") {
		encodedClaim = strings.TrimSuffix(encodedClaim, "=")
	}

	accountID, err := GetAccountID("." + encodedClaim + ".")
	assert.NilError(t, err, "Error getting Account ID from Valid Token")
	assert.Equal(t, 1, accountID, "Wrong accountID returned")
}

func TestGetAccountName(t *testing.T) {
	testClaim := ClaimSet{
		Subject: "testSubject",
	}
	claimAsJSON, _ := json.Marshal(testClaim)
	encodedClaim := base64.URLEncoding.EncodeToString(claimAsJSON)
	for strings.HasSuffix(string(encodedClaim), "=") {
		encodedClaim = strings.TrimSuffix(encodedClaim, "=")
	}

	accountName, err := GetAccountName("." + encodedClaim + ".")
	assert.NilError(t, err, "Error getting AccountName from Valid Token")
	assert.Equal(t, "testSubject", accountName, "Wrong accountName returned")
}
