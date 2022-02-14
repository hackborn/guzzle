package main

import (
	"os"
)

func run(cfg Cfg) (StepOutput, error) {
	output := StepOutput{}
	err := os.MkdirAll(cfg.Output, os.ModePerm)
	if err != nil {
		return output, err
	}
	steps, err := buildSteps(cfg)
	if err != nil {
		return output, err
	}
	p := StepParams{Cfg: cfg, Output: &output}
	err = runSteps(p, steps)
	return output, err
}

func runSteps(p StepParams, steps []Step) error {
	for _, step := range steps {
		err := step.Run(p)
		if err != nil {
			return err
		}
	}
	return nil
}
