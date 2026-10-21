// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

var ErrTooManyImportsInFile = errors.New("file exceeds the per-file import bound")

type FileImports struct{ imports []GoImportPath }

func ParseFileImports(imports []GoImportPath, limits Limits) (FileImports, error) {
	if len(imports) > limits.MaxImportsPerFile {
		return FileImports{}, fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInFile, len(imports), limits.MaxImportsPerFile)
	}
	return FileImports{imports: slices.Clone(imports)}, nil
}

func (parsed FileImports) ImportPaths() []GoImportPath { return slices.Clone(parsed.imports) }

func (parsed FileImports) importsAbsentFrom(other FileImports) []GoImportPath {
	present := make(map[GoImportPath]bool, len(other.imports))
	for _, imported := range other.imports {
		present[imported] = true
	}
	var absent []GoImportPath
	for _, imported := range parsed.imports {
		if !present[imported] {
			absent = append(absent, imported)
		}
	}
	return absent
}
