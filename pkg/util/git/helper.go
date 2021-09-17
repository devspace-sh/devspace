package git

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
)

// GetBranch retrieves the current HEADs name
func GetBranch(localPath string) (string, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return "", errors.Wrap(err, "git open")
	}

	head, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "get head")
	}

	return head.Name().Short(), nil
}

// GetHash retrieves the current HEADs hash
func GetHash(localPath string) (string, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		// last resort, try with cli
		if isGitCommandAvailable() {
			out, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
			if err != nil {
				return "", errors.Errorf("Error running 'git rev-parse HEAD': %v -> %s", err, string(out))
			}

			return strings.TrimSpace(string(out)), nil
		}

		return "", errors.Wrap(err, "git open")
	}

	head, err := repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "get head")
	}

	return head.Hash().String(), nil
}

// GetRemote retrieves the remote origin
func GetRemote(localPath string) (string, error) {
	_, err := os.Stat(localPath + "/.git")
	if err != nil {
		return "", err
	}

	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return "", errors.Wrap(err, "git open")
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", errors.Wrap(err, "get remotes")
	}

	if len(remotes) == 0 {
		return "", errors.Errorf("Couldn't determine git remote in %s", localPath)
	}

	urls := remotes[0].Config().URLs
	if len(urls) == 0 {
		return "", errors.New("No remotes found")
	}

	return urls[0], nil
}
