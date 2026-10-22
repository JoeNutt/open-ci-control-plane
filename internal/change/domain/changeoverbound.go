// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

type ChangeOverBound struct {
	Head   CommitSHA
	Breach error
}

func (over ChangeOverBound) Error() string { return over.Breach.Error() }

func (over ChangeOverBound) Unwrap() error { return over.Breach }

var errUnnamedChangeBound = errors.New("change exceeds a change bound")

var changeBoundBreaches = []error{
	ErrTooManyChangedFiles, ErrFilePatchTooLarge, ErrDocumentTooLarge, ErrPathTooLong, ErrPathTooDeep,
}

const (
	overBoundTitle  = "Change over a bound"
	overBoundLayout = "Refused: %v. No policy was evaluated over this change."
)

func CheckRunForBreach(over ChangeOverBound, id DecisionID) CheckRun {
	return CheckRun{
		Name:       checkRunNameFor(PolicyFamilyStructural),
		Conclusion: ConclusionFailure,
		Title:      overBoundTitle,
		Headline:   fmt.Sprintf(overBoundLayout, breachedBoundOf(over.Breach)),
		DecisionID: id,
	}
}

func breachedBoundOf(breach error) error {
	for _, bound := range changeBoundBreaches {
		if errors.Is(breach, bound) {
			return bound
		}
	}
	return errUnnamedChangeBound
}
