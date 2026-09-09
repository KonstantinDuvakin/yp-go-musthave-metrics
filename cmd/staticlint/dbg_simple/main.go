package main

import (
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/simple"
)

func main() { multichecker.Main(simple.Analyzers[0].Analyzer) }
