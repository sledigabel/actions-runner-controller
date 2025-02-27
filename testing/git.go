package testing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitRepo struct {
	Dir           string
	Name          string
	CommitMessage string
	Contents      map[string][]byte
}

func (g *GitRepo) Sync(ctx context.Context) error {
	repoName := g.Name
	if repoName == "" {
		return errors.New("missing git repo name")
	}

	repoURL := fmt.Sprintf("git@github.com:%s.git", repoName)

	if g.Dir == "" {
		return errors.New("missing git dir")
	}

	dir, err := filepath.Abs(g.Dir)
	if err != nil {
		return fmt.Errorf("error getting abs path for %q: %w", g.Dir, err)
	}

	if _, err := g.combinedOutput(g.gitCloneCmd(ctx, repoURL, dir)); err != nil {
		return err
	}

	for path, content := range g.Contents {
		absPath := filepath.Join(dir, path)

		if err := os.WriteFile(absPath, content, 0755); err != nil {
			return fmt.Errorf("error writing %s: %w", path, err)
		}

		if _, err := g.combinedOutput(g.gitAddCmd(ctx, dir, path)); err != nil {
			return err
		}
	}

	if _, err := g.combinedOutput(g.gitDiffCmd(ctx, dir)); err != nil {
		if _, err := g.combinedOutput(g.gitCommitCmd(ctx, dir, g.CommitMessage)); err != nil {
			return err
		}

		if _, err := g.combinedOutput(g.gitPushCmd(ctx, dir)); err != nil {
			return err
		}
	}

	return nil
}

func (g *GitRepo) gitCloneCmd(ctx context.Context, repo, dir string) *exec.Cmd {
	return exec.CommandContext(ctx, "git", "clone", repo, dir)
}

func (g *GitRepo) gitDiffCmd(ctx context.Context, dir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", "diff", "--exit-code", "--cached")
	cmd.Dir = dir
	return cmd
}

func (g *GitRepo) gitAddCmd(ctx context.Context, dir, path string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", "add", path)
	cmd.Dir = dir
	return cmd
}

func (g *GitRepo) gitCommitCmd(ctx context.Context, dir, msg string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", msg)
	cmd.Dir = dir
	return cmd
}

func (g *GitRepo) gitPushCmd(ctx context.Context, dir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", "push", "origin", "main")
	cmd.Dir = dir
	return cmd
}

func (g *GitRepo) combinedOutput(cmd *exec.Cmd) (string, error) {
	o, err := cmd.CombinedOutput()
	if err != nil {
		args := append([]string{}, cmd.Args...)
		args[0] = cmd.Path

		cs := strings.Join(args, " ")
		s := string(o)
		g.errorf("%s failed with output:\n%s", cs, s)

		return s, err
	}

	return string(o), nil
}

func (g *GitRepo) errorf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
}
