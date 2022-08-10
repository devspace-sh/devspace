package git

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	plumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

// GoGitRepository holds the information about a repository
type GoGitRepository struct {
	LocalPath string
	RemoteURL string
}

// NewGoGitRepository creates a new git repository struct with the given parameters
func NewGoGitRepository(localPath string, remoteURL string) *GoGitRepository {
	return &GoGitRepository{
		LocalPath: localPath,
		RemoteURL: remoteURL,
	}
}

// Update pulls the repository or clones it into the local path
func (gr *GoGitRepository) Update(merge bool) error {
	// Check if repo already exists
	_, err := os.Stat(gr.LocalPath + "/.git")
	if err != nil {
		// Create local path
		err := os.MkdirAll(gr.LocalPath, 0755)
		if err != nil {
			return err
		}

		// Check
		// Clone into folder
		_, err = git.PlainClone(gr.LocalPath, false, &git.CloneOptions{
			URL: gr.RemoteURL,
		})
		if err != nil {
			return err
		}

		return nil
	}

	// Open existing repo
	repo, err := git.PlainOpen(gr.LocalPath)
	if err != nil {
		return err
	}

	// Pull or fetch?
	if merge {
		repoWorktree, err := repo.Worktree()
		if err != nil {
			return err
		}

		// Make sure main is checked out
		err = repoWorktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/main"),
			Create: false,
		})
		if err != nil {
			return err
		}

		err = repoWorktree.Pull(&git.PullOptions{
			RemoteName: "origin",
		})
		if err != git.NoErrAlreadyUpToDate && err != nil {
			return err
		}
	} else {
		err = repo.Fetch(&git.FetchOptions{
			RemoteName: "origin",
		})
		if err != git.NoErrAlreadyUpToDate && err != nil {
			return err
		}
	}

	return nil
}

// Checkout certain tag, branch or hash
func (gr *GoGitRepository) Checkout(tag, branch, revision string) error {
	r, err := git.PlainOpen(gr.LocalPath)
	if err != nil {
		return err
	}

	// Resolve to the correct hash
	var hash *plumbing.Hash
	if tag != "" {
		hash, err = r.ResolveRevision(plumbing.Revision(fmt.Sprintf("refs/tags/%s", tag)))
		if err != nil {
			return errors.Errorf("Error resolving tag revision: %v", err)
		}
	} else if branch != "" {
		remoteRef, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", branch)), true)
		if err != nil {
			return errors.Errorf("Error resolving branch revision: %v", err)
		}

		newRef := plumbing.NewHashReference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)), remoteRef.Hash())
		err = r.Storer.SetReference(newRef)
		if err != nil {
			return err
		}

		// Checkout the branch
		w, err := r.Worktree()
		if err != nil {
			return err
		}

		return w.Checkout(&git.CheckoutOptions{
			Branch: newRef.Name(),
			Create: false,
		})
	} else if revision != "" {
		h := plumbing.NewHash(revision)
		hash = &h
	} else {
		return errors.New("Tag, branch or hash has to be defined")
	}

	// Checkout the hash
	w, err := r.Worktree()
	if err != nil {
		return err
	}

	return w.Checkout(&git.CheckoutOptions{
		Hash:  *hash,
		Force: true,
	})
}
