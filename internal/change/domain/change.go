// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type Change struct {
	Files     BoundedFiles
	Structure StructuralFacts
}

type ImportedFile struct {
	File File
	Base FileImports
	Head FileImports
}

func (imported ImportedFile) existsAtHead() bool { return imported.File.Action.ExistsAtHead() }

func (imported ImportedFile) headImport() Import {
	return Import{
		Path:  imported.File.Path,
		State: ImportStateResolved,
		Refs:  asWritten(imported.Head.ImportPaths()),
	}
}

func (imported ImportedFile) edgesAdded() []ModuleEdge {
	return imported.edgesTo(imported.Head.importsAbsentFrom(imported.Base))
}

func (imported ImportedFile) edgesRemoved() []ModuleEdge {
	return imported.edgesTo(imported.Base.importsAbsentFrom(imported.Head))
}

func (imported ImportedFile) edgesTo(targets []GoImportPath) []ModuleEdge {
	edges := make([]ModuleEdge, 0, len(targets))
	for _, target := range targets {
		edges = append(edges, ModuleEdge{From: imported.File.Path, To: target.String()})
	}
	return edges
}

func asWritten(imports []GoImportPath) Declared[string] {
	written := make(Declared[string], 0, len(imports))
	for _, imported := range imports {
		written = append(written, imported.String())
	}
	return written
}
