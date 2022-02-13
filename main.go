package main

import (
//	"fmt"
)

func main() {
	cfg, err := LoadCfgLocal("cfg.json")
	checkErr(err)
	run(cfg)
}
