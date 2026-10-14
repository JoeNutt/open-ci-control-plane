// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "errors"

type Conclusion string

const (
	ConclusionSuccess Conclusion = "success"
	ConclusionFailure Conclusion = "failure"
	ConclusionNeutral Conclusion = "neutral"
)

var declaredConclusions = []Conclusion{
	ConclusionSuccess,
	ConclusionFailure,
	ConclusionNeutral,
}

var ErrUnknownConclusion = errors.New("unknown check run conclusion")

func ParseConclusion(name string) (Conclusion, error) {
	return parseDeclaration(name, declaredConclusions, ErrUnknownConclusion)
}
