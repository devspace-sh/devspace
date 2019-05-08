package dependency

import (
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/hash"
)

func TestHash(t *testing.T) {
	hash, err := hash.Directory("/Users/fabiankramm/Programmieren/go-workspace/src/gitlab.com/covexo/devspace-cloud")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatal(hash)
}
