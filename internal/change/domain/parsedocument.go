// SPDX-License-Identifier: GPL-3.0-or-later

package domain

func ParseDocument(pullRequest PullRequest, files []File) (Document, error) {
	totals := Totals{Files: len(files)}
	for _, file := range files {
		totals.Additions += file.Additions
		totals.Deletions += file.Deletions
		totals.PatchBytes += file.PatchBytes
	}
	return Document{PR: pullRequest, Diff: Diff{Totals: totals}}, nil
}
