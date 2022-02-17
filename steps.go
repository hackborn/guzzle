package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Step interface {
	Run(StepParams) error
}

type StepParams struct {
	Cfg              Cfg
	CommonCodeFolder string
	Output           *StepOutput
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
	types := make(map[string]AuditRow)
	f := os.DirFS(s.Folder)
	fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if !d.IsDir() {
			var size int64 = 0
			if fi, err := d.Info(); err == nil {
				size = fi.Size()
			}
			ext := strings.ToLower(filepath.Ext(path))
			if row, ok := types[ext]; ok {
				row.Count += 1
				row.Size += size
				types[ext] = row
			} else {
				types[ext] = AuditRow{Name: ext, Size: size, Count: 1}
			}
		}
		return nil
	})
	var rows []AuditRow
	for _, r := range types {
		rows = append(rows, r)
	}
	sort.Sort(SortAuditRowBySize(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		fmt.Println(rows[i])
	}
	return nil
}

type AuditRow struct {
	Name  string
	Size  int64
	Count int
}

type SortAuditRowBySize []AuditRow

func (s SortAuditRowBySize) Len() int {
	return len(s)
}
func (s SortAuditRowBySize) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SortAuditRowBySize) Less(i, j int) bool {
	if s[i].Size < s[j].Size {
		return true
	}
	return s[i].Size < s[j].Size
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Dir = s.LocalFolder
	out, err := cmd.Output()
	if err != nil {
		// I don't really know what to do with checkout errors
		// Some repos do not seem to have tags that correspond
		// to the tag in the go.mod.
		p.AddError(fmt.Errorf("%w out: %v err: %v", err, string(out), string(stderr.Bytes())))
	}
	return nil
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

// CopyStep performs a copy.
type CopyStep struct {
	Src string
	Dst string
}

func (s CopyStep) Run(p StepParams) error {
	fmt.Println("copy", s.Src, "to", s.Dst)
	return fsCopyDir(s.Src, s.Dst)
}

// DeleteGitStep deletes .git related data.
type DeleteGitStep struct {
	Folder string
}

func (s DeleteGitStep) Run(p StepParams) error {
	ext := gitDeletes
	ext = append(ext, codeDeletes...)
	step := DeleteStep{Folder: s.Folder, Ext: ext, Recurse: true}
	return step.Run(p)
}

// DeleteUnityStep deletes Unity-related data.
type DeleteUnityStep struct {
	Folder string
}

func (s DeleteUnityStep) Run(p StepParams) error {
	ext := unityDeletes
	ext = append(ext, mediaDeletes...)
	// Ton of stuff with no extension, as far as I can tell it's junk
	ext = append(ext, "")
	step := DeleteStep{Folder: s.Folder, Ext: ext, Recurse: true}
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
		if !d.IsDir() && s.needsDelete(path) {
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
	ext := strings.ToLower(filepath.Ext(path))
	for _, cmp := range s.Ext {
		if ext == cmp {
			return true
		}
	}
	return false
}

// DeleteEmptyFoldersStep deletes any folders with no contents.
type DeleteEmptyFoldersStep struct {
	Folder     string
	IncludeGit bool // A hack to also include the .git folder which should have been accounted for in the delete git step but wasn't.
}

func (s DeleteEmptyFoldersStep) Run(p StepParams) error {
	fmt.Println("delete empty folders", s.Folder)
	more, err := s.deleteOne(p)
	for more == true && err == nil {
		more, err = s.deleteOne(p)
	}
	return err
}

// deleteOne deletes any empty folders it finds, returning
// true if it deleted something.
func (s DeleteEmptyFoldersStep) deleteOne(p StepParams) (bool, error) {
	parent := s.Folder
	f := os.DirFS(parent)
	ans := false
	err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if ans == true {
			return fs.SkipDir
		}
		base := filepath.Base(path)
		fullpath := filepath.Join(parent, path)
		if s.IncludeGit == true && base == ".git" {
			fmt.Println("Delete", fullpath)
			ans = true
			return os.RemoveAll(fullpath)
		}
		//		ok, err := fsDirEmpty(f, path)
		//		fmt.Println("isempty", path, "ok", ok, "err", err)
		if s.isEmpty(f, path) {
			fmt.Println("Delete", fullpath)
			ans = true
			return os.Remove(fullpath)
		}
		return nil
	})
	return ans, err
}

func (s DeleteEmptyFoldersStep) isEmpty(f fs.FS, path string) bool {
	ok, err := fsDirEmpty(f, path)
	if err != nil {
		err = fmt.Errorf("path: %v, err: %w", path, err)
	}
	checkErr(err)
	return ok
}

// ------------------------------------------------------------
// CONST and VAR

var (
	gitDeletes   = []string{`.git`, `.github`, `.gitignore`, `.gitattributes`}
	codeDeletes  = []string{`.sig`, `.dbg`, `.targets`, `.pri`, `.pack`, `.props`, `.user`, `.zip`, `.p7s`, `.pdb`, `.config`, `.sample`, `.bat`, `.idx`, `.json`, `.xcworkspacedata`, `.name`, `.pro`}
	unityDeletes = []string{`.doc`, `.rendertexture`, `.pdb`, `.meta`, `.unity`, `.unitypackage`, `.prefab`, `.aar`, `.pak`, `.dat`, `.info`, `.strings`, `.mat`, `.cubemap`, `.anim`, `.guiskin`, `.shadervariants`, `.shadergraph`, `.7z`, `.asset`, `.bin`, `.physicmaterial`, `.rsp`, `.example`, `.modulemap`, `.pem`, `.colors`, `.touchosc`, `.bytes`, `.asmref`, `.gradle`, `.exp`, `.iuml`, `.savedsearch`, `.editorconfig`, `.appxmanifest`, `.blend`}
	mediaDeletes = []string{`.ai`, `.dae`, `.exr`, `.fbx`, `.hdr`, `.jpg`, `.nib`, `.pdf`, `.png`, `.psd`, `.tga`, `.tif`, `.ttf`, `.gltf`, `.glb`, `.wav`, `.otf`, `.txt`, `.xml`, `.icns`, `.rtf`, `.puml`, `.csv`, `.md`, `.xaml`}

	errSshNotValid = fmt.Errorf("SSH not valid")

	errDeletedOne = fmt.Errorf("Deleted one")
)
