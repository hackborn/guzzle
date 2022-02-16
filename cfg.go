package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Cfg struct {
	Output        string         `json:"output,omitempty"`
	RepoFormat    string         `json:"repo_format,omitempty"`
	RepoLanguage  string         `json:"repo_language,omitempty"`
	Repos         []Repo         `json:"repos,omitempty"`
	RepoRedirects []RepoRedirect `json:"repo_redirects,omitempty"`
}

type Repo struct {
	Name     string     `json:"name,omitempty"`
	Language string     `json:"language,omitempty"`
	Copy     []RepoCopy `json:"copy,omitempty"`
}

func (r Repo) RepoCopyFrom(repo string) *RepoCopy {
	for _, rc := range r.Copy {
		if rc.From == repo {
			return &rc
		}
	}
	return nil
}

type RepoCopy struct {
	From string   `json:"from,omitempty"`
	To   []string `json:"to,omitempty"`
}

type RepoRedirect struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
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

func (c Cfg) RemoteRepoHttps(repo string) string {
	return c.formatGitHttps(repo)
}

func (c Cfg) RemoteRepoSsh(repo string) string {
	return c.formatGitSsh(repo)
}

func (c Cfg) LocalRepo(repo string) string {
	pos := strings.LastIndex(repo, "/")
	if pos <= 0 {
		return ""
	}
	return filepath.Join(c.Output, repo[pos+1:])
}

func (c Cfg) GetRedirect(repo string) string {
	for _, d := range c.RepoRedirects {
		if d.From == repo {
			return d.To
		}
	}
	return repo
}

func (c Cfg) formatGitHttps(s string) string {
	return "https://" + s
}

func (c Cfg) formatGitSsh(s string) string {
	return "git@" + strings.Replace(s, "/", ":", 1) + ".git"
}
