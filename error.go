package main

import (
	"fmt"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func wrapErr(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("err %w output: %v", err, msg)
}
