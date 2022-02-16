package main

import (
	"os"
	"path/filepath"
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
	commonCodeFolder, err := makeCommonCode(cfg.Output)
	if err != nil {
		return output, err
	}
	p.CommonCodeFolder = commonCodeFolder
	err = runSteps(p, steps)
	return output, err
}

func makeCommonCode(outputFolder string) (string, error) {
	dst := filepath.Join(outputFolder, "Common Code")
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return "", err
	}
	return dst, nil
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
