// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ShadowedRun struct {
	Plan       DispatchedPlan
	IsReported bool
}

type ShadowWindow struct {
	Runs    int
	Gaps    int
	Escapes int
}

func ShadowWindowOf(mode ShadowedMode, runs []ShadowedRun, escapes []Escape) ShadowWindow {
	var window ShadowWindow
	for _, run := range runs {
		if !run.Plan.Modes.isShadowing(mode) {
			continue
		}
		window.Runs++
		if !run.isEvidence() {
			window.Gaps++
		}
	}
	for _, escape := range escapes {
		if escape.Mode == mode {
			window.Escapes++
		}
	}
	return window
}

func (run ShadowedRun) isEvidence() bool {
	return run.Plan.Recorded.IsRecorded && run.IsReported
}

var testModeOfShadowed = map[ShadowedMode]func(PlanModes) TestMode{
	ShadowedSelection:    func(modes PlanModes) TestMode { return modes.Selection },
	ShadowedMinimisation: func(modes PlanModes) TestMode { return modes.Minimisation },
}

func (modes PlanModes) isShadowing(mode ShadowedMode) bool {
	testMode, isKnown := testModeOfShadowed[mode]
	return isKnown && testMode(modes) == TestModeShadow
}
