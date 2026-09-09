package main

import (
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() { multichecker.Main(errcheck.Analyzer) }
