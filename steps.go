package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Step interface {
	Run(StepParams) error
}

type StepParams struct {
	Cfg    Cfg
	Output *StepOutput
}

func (p StepParams) AddError(err error) {
	if p.Output != nil {
		p.Output.Errors = append(p.Output.Errors, err)
	}
}

type StepOutput struct {
	Errors []error
}

// AuditStep performs an audit of file types in a folder.
type AuditStep struct {
	Folder string
}

func (s AuditStep) Run(p StepParams) error {
	fmt.Println("audit", s.Folder)
	types := make(map[string]int)
	f := os.DirFS(s.Folder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if !d.IsDir() {
			ext := filepath.Ext(path)
			if count, ok := types[ext]; ok {
				types[ext] = count + 1
			} else {
				types[ext] = 1
			}
		}
		return nil
	})
	fmt.Println(types)
	return nil
}

// CheckoutStep performs a git checkout.
type CheckoutStep struct {
	LocalFolder string
	Commit      string
}

func (s CheckoutStep) Run(p StepParams) error {
	fmt.Println("git checkout", s.Commit)
	fmt.Println("\t", s.LocalFolder)
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

func (s CloneStep) Run(p StepParams) error {
	// This is more complicated than I'd like it because I don't
	// know how I can access each repo.
	err, redirect := s.tryClone(p, p.Cfg.RemoteRepoSsh(s.Repo))
	if err == nil {
		return nil
	}
	err, redirect = s.tryClone(p, p.Cfg.RemoteRepoHttps(s.Repo))
	if err == nil {
		return nil
	}
	err, _ = s.tryClone(p, redirect)
	return err
	/*
		fmt.Println("git clone", s.Repo, s.LocalFolder)
		cmd := exec.Command("git", "clone", s.Repo, s.LocalFolder)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			fmt.Println("done printing lines", string(stderr.Bytes()))
		}
		return wrapErr(err, string(out))
	*/
}

func (s CloneStep) tryClone(p StepParams, repo string) (error, string) {
	// This is more complicated than I'd like it because I don't
	// know how I can access each repo.
	redirect := repo
	fmt.Println("git clone", repo, s.LocalFolder)
	cmd := exec.Command("git", "clone", repo, s.LocalFolder)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	msg := string(stderr.Bytes())
	if err != nil {
		if strings.Contains(msg, `ssh: connect to host`) && strings.Contains(msg, `fatal: Could not read from remote repository.`) {
			return errSshNotValid, redirect
		}
		r := s.getRedirect(msg)
		if r != "" {
			redirect = r
		}
	}
	return wrapErr(err, string(out)+" "+msg), redirect
}

func (s CloneStep) getRedirect(str string) string {
	pre := `remote: Use 'git clone `
	suf := `' instead`
	if strings.Contains(str, pre) && strings.Contains(str, suf) {
		pi := strings.Index(str, pre)
		si := strings.Index(str, suf)
		return str[pi+len(pre) : si]
	}
	return ""
}

// CloneOrPullStep performs a git pull if the local folder exists,
// otherwise a git clone.
type CloneOrPullStep struct {
	Repo        string
	LocalFolder string
}

func (s CloneOrPullStep) Run(p StepParams) error {
	if _, err := os.Stat(s.LocalFolder); os.IsNotExist(err) {
		return CloneStep{Repo: s.Repo, LocalFolder: s.LocalFolder}.Run(p)
	} else {
		return PullStep{Repo: s.Repo, LocalFolder: s.LocalFolder}.Run(p)
	}
}

// PullStep performs a git pull.
type PullStep struct {
	Repo        string
	LocalFolder string
}

func (s PullStep) Run(p StepParams) error {
	// Git pull is happening without a current branch.
	// In practise, don't think we actually should have
	// the git pull step, we're always getting a specific commit.
	return nil
	/*
		fmt.Println("git pull")
		fmt.Println("\t", s.LocalFolder)
		cmd := exec.Command("git", "pull")
		cmd.Dir = s.LocalFolder
		out, err := cmd.Output()
		return wrapErr(err, string(out))
	*/
}

// DeleteGitStep deletes .git related data.
type DeleteGitStep struct {
	Folder string
	Ext    []string
}

func (s DeleteGitStep) Run(p StepParams) error {
	step := DeleteStep{Folder: s.Folder, Ext: gitDeletes, Recurse: true}
	return step.Run(p)
}

// DeleteStep deletes all files and folders by extension.
type DeleteStep struct {
	Folder  string
	Ext     []string
	Recurse bool
}

func (s DeleteStep) Run(p StepParams) error {
	var err error
	f := os.DirFS(s.Folder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() && !s.Recurse {
			return fs.SkipDir
		}
		if s.needsDelete(path) {
			abs := filepath.Join(s.Folder, path)
			fmt.Println("delete ", abs)
			if d.IsDir() {
				err = mergeErr(err, os.RemoveAll(abs))
			} else {
				err = mergeErr(err, os.Remove(abs))
			}
		}
		return nil
	})
	return err
}

func (s DeleteStep) needsDelete(path string) bool {
	ext := filepath.Ext(path)
	for _, cmp := range s.Ext {
		if ext == cmp {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------
// CONST and VAR

var (
	gitDeletes = []string{`.git`, `.github`, `.gitignore`}

	errSshNotValid = fmt.Errorf("SSH not valid")
)
