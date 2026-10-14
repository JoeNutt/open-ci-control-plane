// SPDX-License-Identifier: GPL-3.0-or-later

package github

type CheckRuns struct {
	settings Settings
}

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	_, err := ParseChangeSource(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{settings: settings}, nil
}
