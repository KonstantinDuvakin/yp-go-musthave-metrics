package main

import (
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() { multichecker.Main(bodyclose.Analyzer) }
