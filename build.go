package main

import (
	"os"
	"strings"
)

func buildSteps(cfg Cfg) ([]Step, error) {
	var steps []Step
	for _, repo := range cfg.Repos {
		remote := cfg.RemoteRepo(repo.Name)
		local := cfg.LocalRepo(repo.Name)
		if local == "" {
			panic("No local folder for repo " + repo.Name)
		}
		// If the local folder exists we pull, otherwise clone.
		if _, err := os.Stat(local); os.IsNotExist(err) {
			steps = append(steps, CloneStep{Repo: remote, LocalFolder: local})
		} else {
			steps = append(steps, PullStep{Repo: remote, LocalFolder: local})
		}
		switch strings.ToLower(repo.Language) {
		case "go":
			steps = append(steps, GoDependencies{OutputFolder: cfg.Output, LocalFolder: local})
		}
	}
	return steps, nil
}
