package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Step interface {
	Run(cfg Cfg) error
}

// CheckoutStep performs a git checkout.
type CheckoutStep struct {
	LocalFolder string
	Commit      string
}

func (s CheckoutStep) Run(cfg Cfg) error {
	fmt.Println("checkout", s.LocalFolder, "commit", s.Commit)
	cmd := exec.Command("git", "checkout", s.Commit)
	cmd.Dir = s.LocalFolder
	out, err := cmd.Output()
	return wrapErr(err, string(out))
}

// CloneStep performs a git clone.
type CloneStep struct {
	Repo        string
	LocalFolder string
}

func (s CloneStep) Run(cfg Cfg) error {
	fmt.Println("clone", s.Repo, "to", s.LocalFolder)
	cmd := exec.Command("git", "clone", s.Repo, s.LocalFolder)
	out, err := cmd.Output()
	return wrapErr(err, string(out))
}

// PullStep performs a git pull.
type PullStep struct {
	Repo        string
	LocalFolder string
}

func (s PullStep) Run(cfg Cfg) error {
	fmt.Println("pull to", s.LocalFolder)
	cmd := exec.Command("git", "pull")
	cmd.Dir = s.LocalFolder
	out, err := cmd.Output()
	return wrapErr(err, string(out))
}

// GoDependencies finds and clones dependencies for Go code.
type GoDependencies struct {
	OutputFolder string
	LocalFolder  string
}

type GoDependency struct {
	Repo     string
	Checkout string // Will either be tags/{version} or the commit sha
}

func (s GoDependencies) Run(cfg Cfg) error {
	fmt.Println("godeps to", s.LocalFolder)
	sums, err := s.gatherSums()
	if err != nil {
		return err
	}
	deps, err := s.gatherDependencies(sums)
	if err != nil {
		return err
	}
	if len(deps) < 1 {
		return nil
	}
	return s.cloneDependencies(cfg, deps)
}

// gatherSums gathers all the .sum files. Currently restricted to
// top level but might expand if there's reason.
func (s GoDependencies) gatherSums() ([]string, error) {
	var sums []string
	f := os.DirFS(s.LocalFolder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		if path == "go.sum" {
			sums = append(sums, path)
		}
		return nil
	})
	return sums, nil
}

// gatherDependencies finds all dependencies for the sum files.
func (s GoDependencies) gatherDependencies(sums []string) ([]GoDependency, error) {
	var deps []GoDependency
	f := os.DirFS(s.LocalFolder)
	for _, path := range sums {
		b, err := fsReadBytes(f, path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			// We're looking for anything where the second field ends in "/go.mod"
			suffix := "/go.mod"
			if len(fields) > 2 && strings.HasSuffix(fields[1], suffix) {
				deps = append(deps, GoDependency{fields[0], s.getCommit(strings.TrimSuffix(fields[1], suffix))})
			}
		}
	}
	return deps, nil
}

// getCommit translates a go.sum commit tag to a git checkout string.
// Formats I'm aware of:
// * version tag: "v1.36.29"
// * commit sha: "v0.0.0-20200922220541-2c3bb06c6054"
func (s GoDependencies) getCommit(commit string) string {
	if !strings.HasPrefix(commit, "v") {
		panic("unknown commit: " + commit)
	}
	split := strings.Split(commit, "-")
	switch len(split) {
	case 1:
		return "tags/" + split[0]
	case 3:
		return split[2]
	}
	panic("unknown commit: " + commit)
}

func (s GoDependencies) cloneDependencies(cfg Cfg, deps []GoDependency) error {
	dst := filepath.Join(s.OutputFolder, "Dependencies")
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	for _, dep := range deps {
		remote := cfg.RemoteRepo(dep.Repo)
		folder := filepath.Join(dst, dep.Repo)
		err = runSteps(cfg, []Step{CloneStep{remote, folder}, CheckoutStep{folder, dep.Checkout}})
		fmt.Println("dep", dep.Repo, dep.Checkout, "err", err)
		if err != nil {
			return wrapErr(err, fmt.Sprintf("%v %v", dep.Repo, dep.Checkout))
		}
	}
	return nil
}
