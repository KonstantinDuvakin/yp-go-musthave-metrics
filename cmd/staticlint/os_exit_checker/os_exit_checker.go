package osexitchecker

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var OsExitChecker = &analysis.Analyzer{
	Name: "osexitchecker",
	Doc:  "check for using os.Exit in main file and package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	isNotMainPkg := pass.Pkg.Name() != "main"
	isTestFile := strings.HasSuffix(pass.Pkg.Path(), ".test")

	if isNotMainPkg || isTestFile {
		return nil, nil
	}

	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fun, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fun.Name.Name != "main" || fun.Recv != nil {
				continue
			}

			ast.Inspect(fun.Body, func(node ast.Node) bool {
				switch x := node.(type) {
				case *ast.CallExpr:
					selExpr, ok := x.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					fn, ok := pass.TypesInfo.ObjectOf(selExpr.Sel).(*types.Func)
					if !ok {
						return true
					}

					if fn.Pkg() != nil && fn.Pkg().Path() == "os" && fn.Name() == "Exit" {
						pass.Reportf(selExpr.Sel.NamePos, "can't call os.Exit in main")
					}
				}
				return true
			})
		}
	}

	return nil, nil
}
