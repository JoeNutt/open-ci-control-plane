// SPDX-License-Identifier: GPL-3.0-or-later

package golang

import (
	"errors"
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

var ErrUnparsableSource = errors.New("the source does not parse as Go")

type Extractor struct{}

func (Extractor) Claims(path string) bool {
	return strings.HasSuffix(path, goSourceSuffix)
}

func (Extractor) ImportsIn(source domain.BoundedSource) ([]domain.GoImportPath, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), unnamedSource, source.Reader(), parser.ImportsOnly)
	if err != nil {
		return nil, ErrUnparsableSource
	}
	imports := make([]domain.GoImportPath, 0, len(parsed.Imports))
	for _, declared := range parsed.Imports {
		imported, err := importPathOf(declared)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imported)
	}
	return imports, nil
}

func importPathOf(declared *ast.ImportSpec) (domain.GoImportPath, error) {
	imported, err := strconv.Unquote(declared.Path.Value)
	if err != nil {
		return domain.GoImportPath{}, fmt.Errorf("%w: an import path is not a string literal: %w",
			ErrUnparsableSource, err)
	}
	parsed, err := domain.ParseGoImportPath(imported)
	if err != nil {
		return domain.GoImportPath{}, fmt.Errorf("%w: %w", ErrUnparsableSource, err)
	}
	return parsed, nil
}
