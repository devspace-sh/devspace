package git

import (
	"io/ioutil"
	"os"
	"testing"
)

const testHash = "e53f405732f27aeeaa04ac07a542372d6f4b1a88"
const testRepo = "https://github.com/rmccue/test-repository.git"

func TestGit(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gitRepo := NewGitRepository(tempDir, testRepo)

	hasUpdate, err := gitRepo.HasUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if hasUpdate == false {
		t.Fatal("hasUpdate returned false")
	}

	updated, err := gitRepo.Update()
	if err != nil {
		t.Fatal(err)
	}
	if updated == false {
		t.Fatal("didnt update")
	}

	updated, err = gitRepo.Update()
	if err != nil {
		t.Fatal(err)
	}
	if updated == true {
		t.Fatal("updated")
	}

	hasUpdate, err = gitRepo.HasUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if hasUpdate == true {
		t.Fatal("hasUpdate returned true")
	}

	remote, err := gitRepo.GetRemote()
	if err != nil {
		t.Fatal(err)
	}
	if remote != testRepo {
		t.Fatalf("Wrong remote, got %s, expected %s", remote, testRepo)
	}

	hash, err := gitRepo.GetHash()
	if err != nil {
		t.Fatal(err)
	}
	if hash != testHash {
		t.Fatalf("Wrong remote, got %s, expected %s", hash, testHash)
	}
}
