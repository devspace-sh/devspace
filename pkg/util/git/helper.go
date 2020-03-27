package git

import (
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"os"
)

// GetHash retrieves the current HEADs hash
func GetHash(localPath string) (string, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
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
