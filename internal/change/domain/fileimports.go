// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

var ErrTooManyImportsInFile = errors.New("file exceeds the per-file import bound")

type FileImports struct{}

func ParseFileImports(imports []string, limits Limits) (FileImports, error) {
	return FileImports{}, ErrTooManyImportsInFile
}
