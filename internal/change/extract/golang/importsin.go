// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const unnamedSource = ""

var ErrUnparsableSource = errors.New("the source does not parse as Go")

func ImportsIn(source domain.BoundedSource) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), unnamedSource, source.Contents(), parser.ImportsOnly)
	if err != nil {
		return nil, ErrUnparsableSource
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, declared := range parsed.Imports {
		imported, err := importPathOf(declared)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imported)
	}
	return imports, nil
}

func importPathOf(declared *ast.ImportSpec) (string, error) {
	imported, err := strconv.Unquote(declared.Path.Value)
	if err != nil {
		return "", fmt.Errorf("%w: an import path is not a string literal: %w",
			ErrUnparsableSource, err)
	}
	return imported, nil
}
