// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

var ErrTooManyImportsInFile = errors.New("file exceeds the per-file import bound")

type FileImports struct{ imports []string }

func ParseFileImports(imports []string, limits Limits) (FileImports, error) {
	if len(imports) > limits.MaxImportsPerFile {
		return FileImports{}, fmt.Errorf("%w: %d imports exceeds %d",
			ErrTooManyImportsInFile, len(imports), limits.MaxImportsPerFile)
	}
	return FileImports{imports: imports}, nil
}

func (parsed FileImports) ImportPaths() []string { return parsed.imports }
