// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const testingStateType = "T"

func (Extractor) DeclaresTests(path string) bool {
	if !strings.HasSuffix(path, "_test.go") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "testdata" || strings.HasPrefix(segment, "_") || strings.HasPrefix(segment, ".") {
			return false
		}
	}
	return true
}

func (Extractor) TestFunctionsIn(source domain.BoundedSource) ([]domain.TestFunction, error) {
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, unnamedSource, source.Reader(), parser.SkipObjectResolution)
	if err != nil {
		return nil, ErrUnparsableSource
	}
	var functions []domain.TestFunction
	for _, declared := range parsed.Decls {
		function, isFunction := declared.(*ast.FuncDecl)
		if !isFunction || !isTopLevelTest(function) {
			continue
		}
		tested, err := testFunctionOf(files, function)
		if err != nil {
			return nil, err
		}
		functions = append(functions, tested)
	}
	return functions, nil
}

func isTopLevelTest(function *ast.FuncDecl) bool {
	return function.Recv == nil &&
		strings.HasPrefix(function.Name.Name, domain.TopLevelTestPrefix) &&
		takesOnlyTestingState(function.Type.Params.List)
}

func takesOnlyTestingState(parameters []*ast.Field) bool {
	if len(parameters) != 1 || len(parameters[0].Names) > 1 {
		return false
	}
	pointer, isPointer := parameters[0].Type.(*ast.StarExpr)
	if !isPointer {
		return false
	}
	selected, isSelector := pointer.X.(*ast.SelectorExpr)
	return isSelector && selected.Sel.Name == testingStateType
}

func testFunctionOf(files *token.FileSet, function *ast.FuncDecl) (domain.TestFunction, error) {
	return domain.ParseTestFunction(function.Name.Name,
		files.Position(function.Pos()).Line, files.Position(function.End()).Line)
}
