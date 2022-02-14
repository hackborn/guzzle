package main

import (
	"fmt"
)

func main() {
	cfg, err := LoadCfgLocal("cfg.json")
	checkErr(err)
	output, err := run(cfg)
	checkErr(err)
	if len(output.Errors) > 0 {
		fmt.Println("There were errors:")
		for _, e := range output.Errors {
			fmt.Println(e)
		}
	}
}
