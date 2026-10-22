// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"log/slog"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type classifiedFailure struct {
	answer answerBody
	level  slog.Level
}

type causedRefusal struct {
	cause   error
	refusal func(failure error) refusal
	level   slog.Level
}

var coverageRefusals = []causedRefusal{
	{cause: domain.ErrUnverifiedIdentityToken, refusal: func(error) refusal { return unverifiedTokenProblem() }, level: slog.LevelWarn},
	{cause: domain.ErrIdentityIssuerUnavailable, refusal: func(error) refusal { return unverifiedTokenProblem() }, level: slog.LevelError},
	{cause: application.ErrForeignRepository, refusal: func(error) refusal { return foreignRepositoryProblem() }, level: slog.LevelWarn},
	{cause: domain.ErrPathOutsideRepository, refusal: func(error) refusal { return unknownPathProblem() }, level: slog.LevelWarn},
	{cause: errUnknownFormat, refusal: func(error) refusal { return fieldProblem{Code: "unknown_format", Message: "x", Field: formatPart} }, level: slog.LevelWarn},
	{cause: domain.ErrMalformedCoverageReport, refusal: func(error) refusal {
		return fieldProblem{Code: "unparseable_artifact", Message: "x", Field: artifactPart}
	}, level: slog.LevelWarn},
	{cause: errMissingPart, refusal: func(failure error) refusal {
		var missing missingPartError
		errors.As(failure, &missing)
		return fieldProblem{Code: "missing_part", Message: "x", Field: missing.part}
	}, level: slog.LevelWarn},
}

func coverageRefusalOf(failure error) classifiedFailure {
	matching := slices.IndexFunc(coverageRefusals, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return classifiedFailure{answer: failedCoverageProblem(), level: slog.LevelError}
	}
	row := coverageRefusals[matching]
	return classifiedFailure{answer: row.refusal(failure), level: row.level}
}
