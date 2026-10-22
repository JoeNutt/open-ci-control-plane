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
	refusal refusal
	level   slog.Level
}

var coverageRefusals = []causedRefusal{
	{cause: domain.ErrUnverifiedIdentityToken, refusal: unverifiedTokenProblem(), level: slog.LevelWarn},
	{cause: domain.ErrIdentityIssuerUnavailable, refusal: unverifiedTokenProblem(), level: slog.LevelError},
	{cause: application.ErrForeignRepository, refusal: foreignRepositoryProblem(), level: slog.LevelWarn},
	{cause: domain.ErrPathOutsideRepository, refusal: unknownPathProblem(), level: slog.LevelWarn},
	{cause: errNoFormatPart, refusal: missingPartProblem(formatPart), level: slog.LevelWarn},
	{cause: errNoArtifactPart, refusal: missingPartProblem(artifactPart), level: slog.LevelWarn},
	{cause: errUnknownFormat, refusal: unknownFormatProblem(), level: slog.LevelWarn},
	{cause: domain.ErrMalformedCoverageReport, refusal: unparseableArtifactProblem(), level: slog.LevelWarn},
}

func coverageRefusalOf(failure error) classifiedFailure {
	matching := slices.IndexFunc(coverageRefusals, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return classifiedFailure{answer: failedCoverageProblem(), level: slog.LevelError}
	}
	row := coverageRefusals[matching]
	return classifiedFailure{answer: row.refusal, level: row.level}
}
