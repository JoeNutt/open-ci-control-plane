// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const (
	goSourceSuffix = ".go"
	unnamedSource  = ""
)

type Extractor struct{}

func (Extractor) Claims(path string) bool {
	return strings.HasSuffix(path, goSourceSuffix)
}

func (Extractor) ImportsIn(source domain.BoundedSource) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), unnamedSource, source.Reader(), parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("%w as Go", domain.ErrUnparsableSource)
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
			domain.ErrUnparsableSource, err)
	}
	return imported, nil
}
