package main

import (
	"fmt"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func mergeErr(a ...error) error {
	for _, e := range a {
		if e != nil {
			return e
		}
	}
	return nil
}

func wrapErr(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("err %w output: %v", err, msg)
}
