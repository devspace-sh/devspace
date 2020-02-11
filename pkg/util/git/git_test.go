package git

import (
	"io/ioutil"
	"os"
	"testing"
)

const testCheckoutHash = "1cc3799959fb8a454b50bb59d0b5d47b78a6d3da"
const testBranch = "newbr"
const testTag = "tag1"
const testRepo = "https://github.com/thockin/test"

func TestGit(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gitRepo := NewGitRepository(tempDir, testRepo)

	err = gitRepo.Update(true)
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Update(false)
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Update(true)
	if err != nil {
		t.Fatal(err)
	}

	remote, err := gitRepo.GetRemote()
	if err != nil {
		t.Fatal(err)
	}
	if remote != testRepo {
		t.Fatalf("Wrong remote, got %s, expected %s", remote, testRepo)
	}

	err = gitRepo.Checkout("", "", testCheckoutHash)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := gitRepo.GetHash()
	if err != nil {
		t.Fatal(err)
	}
	if hash != testCheckoutHash {
		t.Fatalf("Wrong remote, got %s, expected %s", hash, testCheckoutHash)
	}

	err = gitRepo.Checkout("", testBranch, "")
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Checkout(testTag, "", "")
	if err != nil {
		t.Fatal(err)
	}
}
