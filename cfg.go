package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Cfg struct {
	Shortcuts    map[string]string `json:"shortcuts,omitempty"`
	Output       string            `json:"output,omitempty"`
	RepoFormat   string            `json:"repo_format,omitempty"`
	RepoLanguage string            `json:"repo_language,omitempty"`
	Repos        []Repo            `json:"repos,omitempty"`
}

type Repo struct {
	Name     string `json:"name,omitempty"`
	Language string `json:"language,omitempty"`
}

func LoadCfgLocal(path string) (Cfg, error) {
	return LoadCfg(os.DirFS("."), path)
}

func LoadCfg(f fs.FS, path string) (Cfg, error) {
	cfg := Cfg{}
	b, err := fsReadBytes(f, path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(b, &cfg)
	if cfg.RepoLanguage != "" {
		for i, r := range cfg.Repos {
			if r.Language == "" {
				r.Language = cfg.RepoLanguage
				cfg.Repos[i] = r
			}
		}
	}
	return cfg, err
}

func (c Cfg) RemoteRepo(repo string) string {
	repo = c.expandShortcut(repo)
	// Nothing fancy here -- just convert to the one known format I use.
	switch c.RepoFormat {
	case "git_ssh":
		return c.formatGitSsh(repo)
	}
	return repo
}

func (c Cfg) LocalRepo(repo string) string {
	pos := strings.LastIndex(repo, "/")
	if pos <= 0 {
		return ""
	}
	return filepath.Join(c.Output, repo[pos+1:])
}

func (c Cfg) expandShortcut(s string) string {
	for k, v := range c.Shortcuts {
		if strings.HasPrefix(s, k) {
			return v + strings.TrimPrefix(s, k)
		}
	}
	return s
}

func (c Cfg) formatGitSsh(s string) string {
	return "git@" + strings.Replace(s, "/", ":", 1) + ".git"
}
