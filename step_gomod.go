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

// GoModStep finds and clones dependencies for Go go.mod files.
type GoModStep struct {
	Repo         Repo
	OutputFolder string
	LocalFolder  string
}

func (s GoModStep) Run(p StepParams) error {
	fmt.Println("gomod to", s.LocalFolder)
	mods, err := s.gatherMods()
	if err != nil {
		return err
	}
	deps, err := s.gatherDependencies(mods)
	if err != nil {
		return err
	}
	if len(deps) < 1 {
		return nil
	}
	return s.processDependencies(p, deps)
}

// gatherMods gathers all the go.mod files. Currently restricted to
// top level but might expand if there's reason.
func (s GoModStep) gatherMods() ([]string, error) {
	var sums []string
	f := os.DirFS(s.LocalFolder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		if path == "go.mod" {
			sums = append(sums, path)
		}
		return nil
	})
	return sums, nil
}

// gatherDependencies finds all dependencies for the mod files.
func (s GoModStep) gatherDependencies(sums []string) (map[string]GoModDependency, error) {
	deps := make(map[string]GoModDependency)
	f := os.DirFS(s.LocalFolder)
	for _, path := range sums {
		b, err := fsReadBytes(f, path)
		if err != nil {
			return nil, err
		}
		lines, err := s.getRequireBlock(b)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			key, dep := makeGoModDependency(line)
			deps[key] = dep
		}
	}
	return deps, nil
}

func (s GoModStep) getRequireBlock(fileBytes []byte) ([]string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(fileBytes))
	inBlock := false
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if inBlock {
			if line == ")" {
				return lines, nil
			}
			lines = append(lines, line)
		} else {
			if line == "require (" {
				inBlock = true
			}
		}
	}
	return lines, fmt.Errorf("no require block in go.mod")
}

// processDependencies processes the dependency lists, which
// means optionally cloning the repo, and then thinning the data.
func (s GoModStep) processDependencies(p StepParams, deps map[string]GoModDependency) error {
	dst := filepath.Join(s.OutputFolder, "Common Code")
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	for key, dep := range deps {
		if !strings.Contains(key, `genproto`) {
			// continue
		}
		remote := dep.Repo
		folder := filepath.Join(dst, dep.Repo+versionSeparator+dep.Version.id)
		checkout := dep.Version.gitCheckout()
		// Clone if needed
		steps := s.makeCloneSteps(p, dep.Repo, dst, remote, folder, checkout)
		//		steps := []Step{CloneStep{remote, folder}, CheckoutStep{folder, checkout}}
		//		steps = []Step{OnPathNotExists(folder, steps)}
		// Thin
		steps = append(steps, s.makeThinningSteps(p, folder)...)
		err = runSteps(p, steps)
		if err != nil {
			err = wrapErr(err, fmt.Sprintf("key %v go.mod %v to %v from repo %v", key, dep.Raw, folder, s.Repo.Name))
			// Useful if you want everyone to complete and see the final errors
			// p.AddError(err)
			return err
		}
	}
	return nil
}

// makeCloneSteps answers a pipeline for cloning the repo
// (or copying it if there's a copy rule).
func (s GoModStep) makeCloneSteps(p StepParams, depRepo, commonCode, remote, folder, checkout string) []Step {
	// Copy if there's a copy rule for this repo
	copy := s.Repo.RepoCopyFrom(depRepo)
	if copy != nil {
		return s.makeCopySteps(p, *copy, depRepo, commonCode, remote, folder)
	}
	// Clone if needed
	steps := []Step{CloneStep{remote, folder}, CheckoutStep{folder, checkout}}
	steps = []Step{OnPathNotExists(folder, steps)}
	return steps
}

// makeCopySteps answers steps a pipeline for copying the
// repo from a local folder.
func (s GoModStep) makeCopySteps(p StepParams, copy RepoCopy, depRepo, commonCode, remote, folder string) []Step {
	steps := []Step{}
	for _, src := range copy.To {
		src, dst, ok := s.makeCopySrcDst(src, commonCode)
		if ok {
			steps = append(steps, CopyStep{src, dst})
		}
	}
	return steps
}

// makeCopySrcDst expands the source and dest strings to
// absolute paths.
// NOTE: This is very much based on my local env. Burning
// through some of these pieces as fast as I can.
// Return true if what will be the new directory does not exist.
func (s GoModStep) makeCopySrcDst(src, dst string) (string, string, bool) {
	gomodpathVar := `$GOMODPATH`
	if strings.HasPrefix(src, gomodpathVar) {
		// Example "$GOMODPATH/cloud.google.com/go/speech"
		trunk := strings.TrimPrefix(src, gomodpathVar)
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			panic("no GOPATH")
		}
		gomodpath := filepath.Join(gopath, "pkg", "mod")
		src = filepath.Join(gomodpath, trunk)
		if fsNotExists(src) {
			panic("no src " + src)
		}
		dst = filepath.Join(dst, trunk)
		if fsExists(dst) {
			return "", "", false
		}
		return src, filepath.Dir(dst), true
	}
	panic("unhandled copy paths src " + src + " dst " + dst)
}

func (s GoModStep) makeThinningSteps(p StepParams, folder string) []Step {
	// Audit
	//	return []Step{AuditStep{Folder: folder}}
	return []Step{DeleteGitStep{Folder: folder}}
}

// ------------------------------------------------------------
// TYPES

type GoModDependency struct {
	Repo    string
	Version GoModVersion // The go.mod version.
	Raw     string       // The raw line from go.sum
}

// makeGoModDependency creates a dependency from a line in the
// go.mod file.
func makeGoModDependency(raw string) (string, GoModDependency) {
	// There need to be at least two fields, and the second
	// needs to start with "v"
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		panic("invalid go.mod entry:" + raw)
	}
	repo := fields[0]
	version := makeGoModVersion(fields[1])
	return repo + versionSeparator + version.id, GoModDependency{Repo: repo, Version: version, Raw: raw}
}

// GoModVersion represents a version from a go.sum file.
type GoModVersion struct {
	Type       GoModVersionType
	SumVersion string // The raw version from the go.sum
	id         string // The unique portion of the version
}

// makeGoVersion translates a go.sum commit tag to a version structure.
// Formats I'm aware of:
// * version tag: "v1.36.29"
// * version tag with incompatible repo structure: "v2.1.0+incompatible"
// * commit sha: "v0.0.0-20200922220541-2c3bb06c6054"
func makeGoModVersion(commit string) GoModVersion {
	if !strings.HasPrefix(commit, "v") {
		panic("unknown commit: " + commit)
	}
	split := strings.Split(commit, "-")
	switch len(split) {
	case 1:
		incompatible := `+incompatible`
		if strings.HasSuffix(split[0], incompatible) {
			return GoModVersion{GoModVersionTag, commit, strings.TrimSuffix(split[0], incompatible)}
		}
		return GoModVersion{GoModVersionTag, commit, split[0]}
	case 3:
		return GoModVersion{GoModVersionCommit, commit, split[2]}
	default:
		panic("unknown commit: " + commit)
	}
}

// gitCheckout answers the git checkout string for this version
func (v GoModVersion) gitCheckout() string {
	switch v.Type {
	case GoModVersionTag:
		return "tags/" + v.id
	case GoModVersionCommit:
		return v.id
	default:
		panic(fmt.Sprintf("unknown version type: %v", v))
	}
}

// ------------------------------------------------------------
// CONST and VAR

type GoModVersionType uint8

const (
	GoModVersionTag    GoModVersionType = 1 << iota // A tag version i.e. "v1.36.29"
	GoModVersionCommit                              // A SHA commit i.e. "v0.0.0-20200922220541-2c3bb06c6054"
)

const (
	versionSeparator = `@`
)
