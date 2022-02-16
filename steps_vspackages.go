package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// VsPackagesStep finds all packages in a visual studio project.
// Initially being written for C#/nuget packages, but we'll see
// what happens when I hit a C++ project.
type VsPackagesStep struct {
	Folder string
}

func (s VsPackagesStep) Run(p StepParams) error {
	fmt.Println("vspackages on", s.Folder)
	projs, err := s.gatherProjs(p)
	if err != nil {
		return err
	}
	//	fmt.Println("PROJS", projs)
	refs, err := s.gatherReferences(projs)
	if err != nil {
		return err
	}
	//	fmt.Println("REFS", refs)
	return s.acquireReferences(p, refs)
}

// gatherProjs gathers all the .csproj files.
func (s VsPackagesStep) gatherProjs(p StepParams) ([]string, error) {
	var projs []string
	f := os.DirFS(s.Folder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if d.IsDir() || s.isTest(path) {
			return nil
		}
		if ".csproj" == strings.ToLower(filepath.Ext(path)) {
			projs = append(projs, path)
		}
		return nil
	})
	return projs, nil
}

// gatherReferences gathers all the PackageReferences in the proj files.
func (s VsPackagesStep) gatherReferences(projs []string) ([]VsPackageReference, error) {
	f := os.DirFS(s.Folder)
	m := make(map[string]struct{})
	var ans []VsPackageReference
	for _, proj := range projs {
		d, err := fsReadBytes(f, proj)
		if err != nil {
			return nil, err
		}
		var project VsProject
		if err = xml.Unmarshal(d, &project); err != nil {
			return nil, err
		}
		for _, item := range project.ItemGroups {
			for _, ref := range item.PackageReferences {
				key := ref.Include + "@" + ref.Version
				if _, ok := m[key]; !ok {
					ans = append(ans, ref)
					m[key] = struct{}{}
				}
			}
		}
	}
	return ans, nil
}

// acquireReferences copies all references to the common code folder.
func (s VsPackagesStep) acquireReferences(p StepParams, refs []VsPackageReference) error {
	home, err := os.UserHomeDir()
	checkErr(err)
	packages := filepath.Join(home, `.nuget`, `packages`)
	// For now we rely on packages being in a common location.
	// This will definitely change as we're working on this.
	for _, ref := range refs {
		include := strings.ToLower(ref.Include)
		src := filepath.Join(packages, include, ref.Version)
		if fsNotExists(src) {
			return fmt.Errorf("vspackages file does not exist: " + src)
		}
		dst := filepath.Join(p.CommonCodeFolder, `nuget`, include)
		checkdst := filepath.Join(dst, ref.Version)
		if fsExists(checkdst) {
			continue
		}
		fmt.Println("copy", src, "to", dst)
		if err = fsCopyDir(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// isTest is a dumb, stupid hardcoded filter for test projects.
func (s VsPackagesStep) isTest(p string) bool {
	p = strings.ToLower(p)
	return strings.HasPrefix(p, "test")
}

// ------------------------------------------------------------
// TYPES

type VsProject struct {
	ItemGroups []VsItemGroup `xml:"ItemGroup"`
}

type VsItemGroup struct {
	PackageReferences []VsPackageReference `xml:"PackageReference"`
}

type VsPackageReference struct {
	Include string `xml:"Include,attr"`
	Version string `xml:"Version,attr"`
}

// ------------------------------------------------------------
// CONST and VAR

var ()
