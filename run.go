package main

import (
	"os"
)

func run(cfg Cfg) error {
	err := os.MkdirAll(cfg.Output, os.ModePerm)
	if err != nil {
		return err
	}
	steps, err := buildSteps(cfg)
	if err != nil {
		return err
	}
	return runSteps(cfg, steps)
}

func runSteps(cfg Cfg, steps []Step) error {
	for _, step := range steps {
		err := step.Run(cfg)
		if err != nil {
			return err
		}
	}
	return nil
}
