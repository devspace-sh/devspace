package git

import (
	"context"
	"github.com/loft-sh/devspace/pkg/util/command"
	"mvdan.cc/sh/v3/expand"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// GitCLIRepository holds the information about a repository
type GitCLIRepository struct {
	LocalPath string
}

// NewGitCLIRepository creates a new git repository struct with the given parameters
func NewGitCLIRepository(ctx context.Context, localPath string) (*GitCLIRepository, error) {
	if !isGitCommandAvailable(ctx) {
		return nil, errors.New("git not found in path. Please make sure you have git installed to clone git dependencies")
	}

	return &GitCLIRepository{
		LocalPath: localPath,
	}, nil
}

func isGitCommandAvailable(ctx context.Context) bool {
	_, err := command.Output(ctx, "", expand.ListEnviron(os.Environ()...), "git", "version")
	return err == nil
}

type CloneOptions struct {
	URL            string
	Tag            string
	Branch         string
	Commit         string
	Args           []string
	DisableShallow bool
}

// Clone pulls the repository or clones it into the local path
func (gr *GitCLIRepository) Clone(ctx context.Context, options CloneOptions) error {
	// Check if repo already exists
	_, err := os.Stat(gr.LocalPath + "/.git")
	if err != nil {
		// Create local path
		err := os.MkdirAll(gr.LocalPath, 0755)
		if err != nil {
			return err
		}

		args := []string{"clone", options.URL, gr.LocalPath}
		if options.Branch != "" {
			args = append(args, "--branch", options.Branch)
		} else if options.Tag != "" {
			args = append(args, "--branch", options.Tag)
		}

		// do a shallow clone by default
		if options.Commit == "" && !options.DisableShallow {
			args = append(args, "--depth", "1")
		}

		args = append(args, options.Args...)
		out, err := command.CombinedOutput(ctx, gr.LocalPath, expand.ListEnviron(os.Environ()...), "git", args...)
		if err != nil {
			return errors.Errorf("Error running 'git %s': %v -> %s", strings.Join(args, " "), err, string(out))
		}

		// checkout the commit if necessary
		if options.Commit != "" {
			out, err := command.CombinedOutput(ctx, gr.LocalPath, expand.ListEnviron(os.Environ()...), "git", "-C", gr.LocalPath, "checkout", options.Commit)
			if err != nil {
				return errors.Errorf("Error running 'git checkout %s': %v -> %s", options.Commit, err, string(out))
			}
		}

		return nil
	}

	// make sure the repo is up-to-date
	if options.Commit == "" {
		out, err := command.CombinedOutput(ctx, gr.LocalPath, expand.ListEnviron(os.Environ()...), "git", "-C", gr.LocalPath, "pull")
		if err != nil {
			return errors.Errorf("Error running 'git pull %s': %v -> %s", options.URL, err, string(out))
		}
	}

	return nil
}
