package generator

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
)

// GitRepository holds the information about a repository
type GitRepository struct {
	LocalPath string
	RemotURL  string
}

// NewGitRepository creates a new git repository struct with the given parameters
func NewGitRepository(localPath string, remoteURL string) *GitRepository {
	return &GitRepository{
		LocalPath: localPath,
		RemotURL:  remoteURL,
	}
}

// GetRemote retrieves the remote origin
func (gr *GitRepository) GetRemote() (string, error) {
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

	return remotes[0].String(), nil
}

// HasUpdate checks if there is an update to the repository
func (gr *GitRepository) HasUpdate() (bool, error) {
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
		return false, err
	}

	return remoteHead.Hash().String() != repoHead.Hash().String(), nil
}

// Update pulls the repository or clones it into the local path
func (gr *GitRepository) Update() (bool, error) {
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
