// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func (Extractor) TestFunctionsIn(source domain.BoundedSource) ([]domain.TestFunction, error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, unnamedSource, source.Reader(), parser.SkipObjectResolution)
	if err != nil {
		return nil, ErrUnparsableSource
	}
	var functions []domain.TestFunction
	for _, declared := range parsed.Decls {
		function, isFunction := declared.(*ast.FuncDecl)
		if !isFunction || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "Test") {
			continue
		}
		params := function.Type.Params.List
		if len(params) != 1 || len(params[0].Names) > 1 {
			continue
		}
		star, isPointer := params[0].Type.(*ast.StarExpr)
		if !isPointer {
			continue
		}
		selector, isSelector := star.X.(*ast.SelectorExpr)
		if !isSelector || selector.Sel.Name != "T" {
			continue
		}
		parsedFunction, err := domain.ParseTestFunction(function.Name.Name,
			files.Position(function.Pos()).Line, files.Position(function.End()).Line)
		if err != nil {
			return nil, err
		}
		functions = append(functions, parsedFunction)
	}
	return functions, nil
}
