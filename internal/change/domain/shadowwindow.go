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
		window.Runs++
		if !run.Plan.Recorded.IsRecorded || !run.IsReported {
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
