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

func (imported ImportedFile) hasErroredEnd() bool {
	return imported.Base.isErrored || imported.Head.isErrored
}

func (imported ImportedFile) headImport() Import {
	if imported.Head.isErrored {
		return Import{Path: imported.File.Path, State: ImportStateErrored}
	}
	return Import{
		Path:  imported.File.Path,
		State: ImportStateResolved,
		Refs:  imported.Head.ImportPaths(),
	}
}

func (imported ImportedFile) edgesAdded() []ModuleEdge {
	return imported.edgesTo(imported.Head.importsAbsentFrom(imported.Base))
}

func (imported ImportedFile) edgesRemoved() []ModuleEdge {
	return imported.edgesTo(imported.Base.importsAbsentFrom(imported.Head))
}

func (imported ImportedFile) edgesTo(targets []string) []ModuleEdge {
	if imported.hasErroredEnd() {
		return nil
	}
	edges := make([]ModuleEdge, 0, len(targets))
	for _, target := range targets {
		edges = append(edges, ModuleEdge{From: imported.File.Path, To: target})
	}
	return edges
}
