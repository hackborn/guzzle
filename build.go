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
		// If the local folder exists we pull, otherwise clone.
		steps = append(steps, CloneOrPullStep{Repo: remote, LocalFolder: local})
		switch strings.ToLower(repo.Language) {
		case "go":
			steps = append(steps, GoModStep{Repo: repo.Name, OutputFolder: cfg.Output, LocalFolder: local})
		}
	}
	return steps, nil
}
