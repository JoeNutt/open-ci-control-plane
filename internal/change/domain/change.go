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
