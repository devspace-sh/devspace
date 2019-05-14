package git

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
)

// Repository holds the information about a repository
type Repository struct {
	LocalPath string
	RemotURL  string
}

// NewGitRepository creates a new git repository struct with the given parameters
func NewGitRepository(localPath string, remoteURL string) *Repository {
	return &Repository{
		LocalPath: localPath,
		RemotURL:  remoteURL,
	}
}

// GetHash retrieves the current HEADs hash
func (gr *Repository) GetHash() (string, error) {
	repo, err := git.PlainOpen(gr.LocalPath)
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
func (gr *Repository) GetRemote() (string, error) {
	_, err := os.Stat(gr.LocalPath + "/.git")
	if err != nil {
		return "", err
	}

	repo, err := git.PlainOpen(gr.LocalPath)
	if err != nil {
		return "", errors.Wrap(err, "git open")
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", errors.Wrap(err, "get remotes")
	}

	if len(remotes) == 0 {
		return "", fmt.Errorf("Couldn't determine git remote in %s", gr.LocalPath)
	}

	urls := remotes[0].Config().URLs
	if len(urls) == 0 {
		return "", errors.New("No remotes found")
	}

	return urls[0], nil
}

// HasUpdate checks if there is an update to the repository
func (gr *Repository) HasUpdate() (bool, error) {
	_, err := os.Stat(gr.LocalPath + "/.git")
	if err != nil {
		return true, nil
	}

	repo, err := git.PlainOpen(gr.LocalPath)
	if err != nil {
		return false, err
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != git.NoErrAlreadyUpToDate && err != nil {
		return false, err
	}

	repoHead, err := repo.Head()
	if err != nil {
		return false, err
	}

	remoteHead, err := repo.Reference("refs/remotes/origin/HEAD", true)
	if err != nil {
		return false, nil
	}

	return remoteHead.Hash().String() != repoHead.Hash().String(), nil
}

// Update pulls the repository or clones it into the local path
func (gr *Repository) Update() (bool, error) {
	_, repoNotFound := os.Stat(gr.LocalPath + "/.git")
	if repoNotFound == nil {
		repo, err := git.PlainOpen(gr.LocalPath)
		if err != nil {
			return false, err
		}

		repoWorktree, err := repo.Worktree()
		if err != nil {
			return false, err
		}

		oldHead, err := repo.Head()
		if err != nil {
			return false, err
		}

		err = repoWorktree.Pull(&git.PullOptions{
			RemoteName: "origin",
		})
		if err != git.NoErrAlreadyUpToDate && err != nil {
			return false, err
		}

		newHead, err := repo.Head()
		if err != nil {
			return false, err
		}

		return oldHead.Hash().String() != newHead.Hash().String(), nil
	}

	// Create local path
	err := os.MkdirAll(gr.LocalPath, 0755)
	if err != nil {
		return false, err
	}

	// Clone into folder
	_, err = git.PlainClone(gr.LocalPath, false, &git.CloneOptions{
		URL: gr.RemotURL,
	})
	if err != nil {
		return false, err
	}

	return true, nil
}
