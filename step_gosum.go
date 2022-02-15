package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// NOTE: This is unfinished and essentially archived. As I was
// working on the project requirements changed and this is no
// longer needed.
// What is currently unaddressed is cloning repos that redirect,
// and from servers that don't support ssh. Maybe that'll get
// resolved in the framework.

// GoSumStep finds and clones dependencies for Go go.sum files.
type GoSumStep struct {
	OutputFolder string
	LocalFolder  string
}

func (s GoSumStep) Run(p StepParams) error {
	fmt.Println("gosum to", s.LocalFolder)
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
	return s.cloneDependencies(p, deps)
}

// gatherSums gathers all the .sum files. Currently restricted to
// top level but might expand if there's reason.
func (s GoSumStep) gatherSums() ([]string, error) {
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
func (s GoSumStep) gatherDependencies(sums []string) (map[string]GoSumDependency, error) {
	deps := make(map[string]GoSumDependency)
	f := os.DirFS(s.LocalFolder)
	for _, path := range sums {
		b, err := fsReadBytes(f, path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			raw := scanner.Text()
			fields := strings.Fields(raw)
			// We're looking for anything where the second field ends in "/go.mod"
			suffix := "/go.mod"
			if len(fields) > 2 && strings.HasSuffix(fields[1], suffix) {
				key, dep := makeGoSumDependency(raw, fields[0], strings.TrimSuffix(fields[1], suffix))
				deps[key] = dep
			}
		}
	}
	return deps, nil
}

func (s GoSumStep) cloneDependencies(p StepParams, deps map[string]GoSumDependency) error {
	dst := filepath.Join(s.OutputFolder, "Common Code")
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	for key, dep := range deps {
		if key != `golang.org/x/xerrors-a985d3407aa7` {
			continue
		}
		remote := dep.Repo
		folder := filepath.Join(dst, dep.Repo+"-"+dep.Version.id)
		err = runSteps(p, []Step{CloneOrPullStep{remote, folder}, CheckoutStep{folder, dep.Version.gitCheckout()}})
		if err != nil {
			err = wrapErr(err, fmt.Sprintf("key %v go.sum %v to %v", key, dep.Raw, folder))
			p.AddError(err)
			//			fmt.Println("dep", dep.Repo, dep.Version.gitCheckout(), "err", err)
			//			return wrapErr(err, fmt.Sprintf("%v %v", dep.Repo, dep.Version.gitCheckout()))
		}
	}
	return nil
}

// ------------------------------------------------------------
// TYPES

type GoSumDependency struct {
	Repo    string
	Version GoSumVersion // The go.sum version.
	Raw     string       // The raw line from go.sum
}

func makeGoSumDependency(raw, repo, commit string) (string, GoSumDependency) {
	dep := GoSumDependency{Repo: formatGoSumRepo(repo), Version: makeGoSumVersion(commit), Raw: raw}
	return dep.Repo + "-" + dep.Version.id, dep
}

// formatRepo translates a go.sum repo path to a cloneable repo name.
// Formats I'm aware of:
// * go.uber.org/goleak
// * github.com/aws/aws-sdk-go
// * github.com/go-playground/assert/v2
func formatGoSumRepo(repo string) string {
	repoFields := strings.Split(repo, `/`)
	switch len(repoFields) {
	case 0, 1:
		panic("invalid repo name " + repo)
	case 2:
		return repoFields[0] + `/` + repoFields[1]
	default:
		return repoFields[0] + `/` + repoFields[1] + `/` + repoFields[2]
	}
}

// GoSumVersion represents a version from a go.sum file.
type GoSumVersion struct {
	Type       GoSumVersionType
	SumVersion string // The raw version from the go.sum
	id         string // The unique portion of the version
}

// makeGoSumVersion translates a go.sum commit tag to a version structure.
// Formats I'm aware of:
// * version tag: "v1.36.29"
// * commit sha: "v0.0.0-20200922220541-2c3bb06c6054"
func makeGoSumVersion(commit string) GoSumVersion {
	if !strings.HasPrefix(commit, "v") {
		panic("unknown commit: " + commit)
	}
	split := strings.Split(commit, "-")
	switch len(split) {
	case 1:
		return GoSumVersion{GoSumVersionTag, commit, split[0]}
	case 3:
		return GoSumVersion{GoSumVersionCommit, commit, split[2]}
	default:
		panic("unknown commit: " + commit)
	}
}

// gitCheckout answers the git checkout string for this version
func (v GoSumVersion) gitCheckout() string {
	switch v.Type {
	case GoSumVersionTag:
		return "tags/" + v.id
	case GoSumVersionCommit:
		return v.id
	default:
		panic(fmt.Sprintf("unknown version type: %v", v))
	}
}

// ------------------------------------------------------------
// CONST and VAR

type GoSumVersionType uint8

const (
	GoSumVersionTag    GoSumVersionType = 1 << iota // A tag version i.e. "v1.36.29"
	GoSumVersionCommit                              // A SHA commit i.e. "v0.0.0-20200922220541-2c3bb06c6054"
)
