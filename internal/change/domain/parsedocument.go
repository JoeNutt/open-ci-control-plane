// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func ParseDocument(pullRequest PullRequest, files []File) (Document, error) {
	return Document{PR: pullRequest, Diff: Diff{
		Paths:  pathsOf(files),
		Totals: totalsOf(files),
	}}, nil
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
