// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"errors"
	"unicode/utf8"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

// GitHub accepts at most 65535 characters in each of output.summary and output.text:
// https://docs.github.com/en/rest/checks/runs
const MaxOutputCharacters = 65535

var ErrOutputTooLong = errors.New("a check run's summary or text is longer than the forge accepts")

func parseVerdict(run domain.CheckRun) (checkRunVerdict, error) {
	output, err := parseOutput(run)
	if err != nil {
		return checkRunVerdict{}, err
	}
	conclusion, err := domain.ParseConclusion(string(run.Conclusion))
	if err != nil {
		return checkRunVerdict{}, err
	}
	return checkRunVerdict{
		Status:     statusCompleted,
		Conclusion: string(conclusion),
		Output:     output,
	}, nil
}

func parseOutput(run domain.CheckRun) (checkRunOutput, error) {
	if run.Title == "" || run.Summary == "" {
		return checkRunOutput{}, ErrIncompleteCheckRun
	}
	if !fitsOutput(run.Summary) || !fitsOutput(run.Text) {
		return checkRunOutput{}, ErrOutputTooLong
	}
	return checkRunOutput{Title: run.Title, Summary: run.Summary, Text: run.Text}, nil
}

func fitsOutput(field string) bool {
	return utf8.RuneCountInString(field) <= MaxOutputCharacters
}
