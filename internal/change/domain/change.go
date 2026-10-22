// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type Change struct {
	Files    BoundedFiles
	Imported []ImportedFile
}

type ImportedFile struct {
	File File
	Head FileImports
}

func (imported ImportedFile) headImport() Import {
	return Import{
		Path:  imported.File.Path,
		State: ImportStateResolved,
		Refs:  imported.Head.ImportPaths(),
	}
}
