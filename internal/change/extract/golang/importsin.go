// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

const unnamedSource = ""

func ImportsIn(source []byte) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), unnamedSource, source, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse the import declarations: %w", err)
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
		return "", fmt.Errorf("unquote an import path: %w", err)
	}
	return imported, nil
}
