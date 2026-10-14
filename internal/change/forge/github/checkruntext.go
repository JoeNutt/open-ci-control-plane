// SPDX-License-Identifier: GPL-3.0-or-later

package github

import "github.com/JoeNutt/open-ci-control-plane/internal/change/domain"

type checkRunTextOutput struct {
	title   string
	summary string
	text    string
}

func checkRunTextFor(run domain.CheckRun, capacity int) (checkRunTextOutput, error) {
	return checkRunTextOutput{title: run.Title}, nil
}
