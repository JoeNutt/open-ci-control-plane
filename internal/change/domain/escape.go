// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ShadowedMode string

const ShadowedSelection ShadowedMode = "selection"

type Escape struct {
	Mode         ShadowedMode
	PullRequest  PullRequestLocator
	Test         TestResult
	ChangedFiles []RepositoryPath
}

func (dispatched DispatchedPlan) EscapesIn(results []TestResult) []Escape {
	var escapes []Escape
	for _, escaped := range dispatched.SelectionEscapesIn(results) {
		escapes = append(escapes, Escape{Mode: ShadowedSelection, PullRequest: dispatched.Locator, Test: escaped, ChangedFiles: dispatched.ChangedFiles})
	}
	return escapes
}
