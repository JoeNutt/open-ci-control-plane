// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"path"
	"slices"
)

func ParseDocument(
	pullRequest PullRequest, bounded BoundedFiles, rules ClassificationRules,
) Document {
	return Document{PR: cloneOf(pullRequest), Diff: diffOf(bounded, rules)}
}

func cloneOf(pullRequest PullRequest) PullRequest {
	pullRequest.Labels = slices.Clone(pullRequest.Labels)
	pullRequest.Approvals = slices.Clone(pullRequest.Approvals)
	return pullRequest
}

func diffOf(bounded BoundedFiles, rules ClassificationRules) Diff {
	paths := pathsOf(bounded.files)
	imports := Declared[Import]{}
	resolution := importResolutionIn(imports)
	return Diff{
		Paths:                 paths,
		Kinds:                 rules.KindsOf(paths),
		Files:                 slices.Clone(bounded.files),
		Totals:                totalsOf(bounded.files),
		ByDir:                 byDirOf(bounded.files),
		NewFiles:              addedPathsOf(bounded.files),
		Imports:               imports,
		UnresolvedImportFiles: resolution.unresolved,
		ResolvedImportShare:   resolution.share,
		ModuleGraphDelta:      graphDeltaFor(resolution),
	}
}

func pathsOf(files []File) Paths {
	return pathsMatching(files, everyFile)
}

func addedPathsOf(files []File) Paths {
	return pathsMatching(files, fileAdded)
}

func pathsMatching(files []File, selected func(File) bool) Paths {
	matching := make(Paths, 0, len(files))
	for _, file := range files {
		if selected(file) {
			matching = append(matching, file.Path)
		}
	}
	return matching
}

func everyFile(File) bool { return true }

func fileAdded(file File) bool { return file.Action == FileActionAdded }

func totalsOf(files []File) Totals {
	totals := Totals{Files: len(files)}
	for _, file := range files {
		totals.Additions += file.Additions
		totals.Deletions += file.Deletions
		totals.PatchBytes += file.PatchBytes
	}
	return totals
}

func byDirOf(files []File) PathsByDir {
	byDir := make(PathsByDir, len(files))
	for _, file := range files {
		directory := path.Dir(file.Path)
		byDir[directory] = append(byDir[directory], file.Path)
	}
	return byDir
}

type importResolution struct {
	unresolved int
	share      float64
}

func importResolutionIn(imports Declared[Import]) importResolution {
	unresolved := 0
	for _, imported := range imports {
		if imported.State != ImportStateResolved {
			unresolved++
		}
	}
	resolution := importResolution{unresolved: unresolved}
	if len(imports) > 0 {
		resolution.share = float64(len(imports)-unresolved) / float64(len(imports))
	}
	return resolution
}

func graphDeltaFor(resolution importResolution) ModuleGraphDelta {
	return ModuleGraphDelta{Incomplete: resolution.unresolved > 0}
}
