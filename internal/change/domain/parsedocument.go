// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"path"
	"slices"
)

func ParseDocument(pullRequest PullRequest, bounded BoundedFiles) Document {
	return Document{PR: cloneOf(pullRequest), Diff: diffOf(bounded)}
}

func cloneOf(pullRequest PullRequest) PullRequest {
	pullRequest.Labels = slices.Clone(pullRequest.Labels)
	pullRequest.Approvals = slices.Clone(pullRequest.Approvals)
	return pullRequest
}

func diffOf(bounded BoundedFiles) Diff {
	return Diff{
		Paths:  pathsOf(bounded.files),
		Files:  slices.Clone(bounded.files),
		Totals: totalsOf(bounded.files),
		ByDir:  byDirOf(bounded.files),
	}
}

func pathsOf(files []File) Paths {
	paths := make(Paths, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

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
