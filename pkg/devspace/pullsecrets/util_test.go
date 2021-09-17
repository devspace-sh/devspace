package pullsecrets

import (
	"testing"

	"gotest.tools/assert"
)

func TestGetRegistryFromImageName(t *testing.T) {
	//Test with official repo
	reg, err := GetRegistryFromImageName("mysql")
	if err != nil {
		t.Fatalf("Error calling GetRegistryFromImageName: %v", err)
	}
	assert.Equal(t, "", reg, "Official repo can't be detected")

	//Test with unofficial repo
	reg, err = GetRegistryFromImageName("reg.example.com/foobar")
	if err != nil {
		t.Fatalf("Error calling GetRegistryFromImageName: %v", err)
	}
	assert.Equal(t, "reg.example.com", reg, "Unofficial repo can't be detected")

	//Test with invalid
	_, err = GetRegistryFromImageName("")
	if err == nil {
		t.Fatalf("No Error calling GetRegistryFromImageName with empty image name")
	}
}
