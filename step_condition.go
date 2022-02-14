package main

import ()

// ------------------------------------------------------------
// MACROS

// OnPathExists performs the steps if the path exists.
func OnPathExists(path string, steps []Step) IfConditionStep {
	fn := func() bool {
		return fsExists(path)
	}
	return IfConditionStep{Condition: fn, Steps: steps}
}

// OnPathNotExists performs the steps if the path does not exist.
func OnPathNotExists(path string, steps []Step) IfConditionStep {
	fn := func() bool {
		return fsNotExists(path)
	}
	return IfConditionStep{Condition: fn, Steps: steps}
}

// ------------------------------------------------------------
// IF-CONDITION-STEP

// IfConditionStep performs a pipeline on the condition.
type IfConditionStep struct {
	Condition ConditionFunc
	Steps     []Step
}

func (s IfConditionStep) Run(p StepParams) error {
	if s.Condition == nil || s.Condition() == false {
		return nil
	}
	return runSteps(p, s.Steps)
}

// ------------------------------------------------------------
// OR-CONDITION-STEP

// OrConditionStep performs a pipeline depending on the condtion.
type OrConditionStep struct {
	Condition  ConditionFunc
	TrueSteps  []Step
	FalseSteps []Step
}

func (s OrConditionStep) Run(p StepParams) error {
	if s.Condition == nil {
		return nil
	}
	if s.Condition() {
		return runSteps(p, s.TrueSteps)
	} else {
		return runSteps(p, s.FalseSteps)
	}
}

// ------------------------------------------------------------
// FUNCS

type ConditionFunc func() bool
