// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type ShadowedMode string

const (
	ShadowedSelection    ShadowedMode = "selection"
	ShadowedMinimisation ShadowedMode = "minimisation"
)

var shadowedModes = []ShadowedMode{ShadowedSelection, ShadowedMinimisation}

var ErrUnknownShadowedMode = errors.New("the mode is not selection or minimisation")

func ParseShadowedMode(stated string) (ShadowedMode, error) {
	if !slices.Contains(shadowedModes, ShadowedMode(stated)) {
		return "", fmt.Errorf("%w: %q", ErrUnknownShadowedMode, stated)
	}
	return ShadowedMode(stated), nil
}

type Escape struct {
	At           time.Time
	Mode         ShadowedMode
	PullRequest  PullRequestLocator
	Test         TestResult
	ChangedFiles []RepositoryPath
}

func (dispatched DispatchedPlan) EscapesIn(results []TestResult) []Escape {
	var escapes []Escape
	for _, shadowed := range dispatched.shadowedSets() {
		for _, escaped := range dispatched.escapesOutside(shadowed.recorded, results) {
			escapes = append(escapes, Escape{
				Mode: shadowed.mode, PullRequest: dispatched.Locator, Test: escaped, ChangedFiles: dispatched.ChangedFiles,
			})
		}
	}
	return escapes
}

type shadowedSet struct {
	mode     ShadowedMode
	recorded []PackageTests
}

func (dispatched DispatchedPlan) shadowedSets() []shadowedSet {
	if !dispatched.Recorded.IsRecorded {
		return nil
	}
	sets := []shadowedSet{{mode: ShadowedSelection, recorded: dispatched.Recorded.Selected}}
	if dispatched.Modes.isShadowing(ShadowedMinimisation) {
		sets = append(sets, shadowedSet{mode: ShadowedMinimisation, recorded: dispatched.Recorded.Minimised})
	}
	return sets
}

func (dispatched DispatchedPlan) escapesOutside(recorded []PackageTests, results []TestResult) []TestResult {
	var escaped []TestResult
	for _, result := range results {
		if result.failedTwice() && !dispatched.wouldHaveRun(recorded, result) {
			escaped = append(escaped, result)
		}
	}
	return escaped
}

func (dispatched DispatchedPlan) wouldHaveRun(recorded []PackageTests, result TestResult) bool {
	return slices.ContainsFunc(recorded, func(entry PackageTests) bool { return entry.runs(result) }) ||
		slices.Contains(dispatched.AlwaysRun, result.Suite.importPath)
}

func (result TestResult) failedTwice() bool {
	return result.Outcome == TestOutcomeFail && result.Rerun == TestOutcomeFail
}
