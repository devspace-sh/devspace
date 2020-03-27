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

func TestGitCliCommit(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gitRepo, err := NewGitCLIRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Clone(CloneOptions{
		URL:    testRepo,
		Commit: testCheckoutHash,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Clone(CloneOptions{
		URL:    testRepo,
		Commit: testCheckoutHash,
	})
	if err != nil {
		t.Fatal(err)
	}

	hash, err := GetHash(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if hash != testCheckoutHash {
		t.Fatalf("Wrong remote, got %s, expected %s", hash, testCheckoutHash)
	}
}

func TestGitCliBranch(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gitRepo, err := NewGitCLIRepository(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Clone(CloneOptions{
		URL:    testRepo,
		Branch: testBranch,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = gitRepo.Clone(CloneOptions{
		URL:            testRepo,
		Branch:         testBranch,
		DisableShallow: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	remote, err := GetRemote(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if remote != testRepo {
		t.Fatalf("Wrong remote, got %s, expected %s", remote, testRepo)
	}
}

func TestGoGit(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	gitRepo := NewGoGitRepository(tempDir, testRepo)
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
	remote, err := GetRemote(tempDir)
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

	hash, err := GetHash(tempDir)
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
