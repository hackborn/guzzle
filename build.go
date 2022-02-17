package main

import (
	"fmt"
	"strings"
)

func buildSteps(cfg Cfg) ([]Step, error) {
	var steps []Step
	for _, repo := range cfg.Repos {
		// Shortcut for disabling repos
		if strings.HasPrefix(repo.Name, "//") {
			fmt.Println("skipping repo", repo.Name)
			continue
		}
		remote := repo.Name
		local := cfg.LocalRepo(repo.Name)
		if local == "" {
			panic("No local folder for repo " + repo.Name)
		}
		// Clone if needed. Once we have a clone we thin out the data,
		// which removes git info. If you want to reclone, you need to
		// manually remove it from the output.
		cloneSteps := []Step{CloneStep{Repo: remote, LocalFolder: local}}
		if repo.Branch != "" {
			cloneSteps = append(cloneSteps, CheckoutStep{Commit: repo.Branch, LocalFolder: local})
		}
		steps = append(steps, []Step{OnPathNotExists(local, cloneSteps)}...)
		// Add generic thinning
		switch strings.ToLower(repo.Language) {
		case "go":
			steps = append(steps, GoModStep{Repo: repo, OutputFolder: cfg.Output, LocalFolder: local})
		case "c#":
			steps = append(steps, AuditStep{Folder: local})
			steps = append(steps, VsPackagesStep{Folder: local})
		default:
			steps = append(steps, AuditStep{Folder: local})
			steps = append(steps, DeleteUnityStep{Folder: local})
		}
		// Remove git data
		steps = append(steps, DeleteGitStep{Folder: local})
		// Tidy
		steps = append(steps, DeleteEmptyFoldersStep{Folder: local, IncludeGit: true})
	}
	return steps, nil
}
