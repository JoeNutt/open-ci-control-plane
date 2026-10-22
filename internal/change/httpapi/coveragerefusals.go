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
	reason string
	cause  error
	level  slog.Level
}

func refusedBeforeUse(refused refusal) classifiedFailure {
	return classifiedFailure{answer: refused, reason: refused.reason(), level: slog.LevelWarn}
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
	{cause: domain.ErrCompressedArtifactTooLarge, refusal: boundProblem(compressedArtifactLimit), level: slog.LevelWarn},
	{cause: domain.ErrDecompressedArtifactTooLarge, refusal: boundProblem(decompressedArtifactLimit), level: slog.LevelWarn},
	{cause: domain.ErrTooManyFileEntries, refusal: boundProblem(fileEntriesLimit), level: slog.LevelWarn},
	{cause: domain.ErrTooManySuites, refusal: boundProblem(suitesLimit), level: slog.LevelWarn},
}

func coverageRefusalOf(failure error) classifiedFailure {
	var overRate application.UploadRateRefusal
	if errors.As(failure, &overRate) {
		return classifiedFailure{answer: rateProblemAfter(overRate.RetryAfter), reason: rateExceededCode, cause: failure, level: slog.LevelWarn}
	}
	matching := slices.IndexFunc(coverageRefusals, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return classifiedFailure{answer: failedCoverageProblem(), reason: failedCoverageCode, cause: failure, level: slog.LevelError}
	}
	row := coverageRefusals[matching]
	return classifiedFailure{answer: row.refusal, reason: row.refusal.reason(), cause: failure, level: row.level}
}
