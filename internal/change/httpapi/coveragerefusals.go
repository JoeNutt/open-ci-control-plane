// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type loggedAnswer struct {
	answer answerBody
	reason string
	cause  error
	level  slog.Level
}

func (answered loggedAnswer) logTo(ctx context.Context, log *slog.Logger, message string) {
	if answered.reason == "" {
		return
	}
	named := []slog.Attr{slog.String(reasonKey, answered.reason)}
	if answered.cause != nil {
		named = append(named, slog.String(causeKey, answered.cause.Error()))
	}
	log.LogAttrs(ctx, answered.level, message, named...)
}

func refusedBeforeUse(refused refusal) loggedAnswer {
	return loggedAnswer{answer: refused, reason: refused.reason(), level: slog.LevelWarn}
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
	{cause: domain.ErrTooManyTestSections, refusal: boundProblem(testSectionsLimit), level: slog.LevelWarn},
	{cause: domain.ErrTooManyLineRecords, refusal: boundProblem(lineRecordsLimit), level: slog.LevelWarn},
	{cause: domain.ErrTooManyIndexedTests, refusal: boundProblem(indexedTestsLimit), level: slog.LevelWarn},
	{cause: domain.ErrTooManyIndexEdges, refusal: boundProblem(indexEdgesLimit), level: slog.LevelWarn},
}

func coverageRefusalOf(failure error) loggedAnswer {
	var overRate application.UploadRateRefusal
	if errors.As(failure, &overRate) {
		return loggedAnswer{answer: rateProblemAfter(overRate.RetryAfter), reason: rateExceededCode, cause: failure, level: slog.LevelWarn}
	}
	row, isMatched := refusalCausing(coverageRefusals, failure)
	if !isMatched {
		return loggedAnswer{answer: failedCoverageProblem(), reason: failedCoverageCode, cause: failure, level: slog.LevelError}
	}
	return loggedAnswer{answer: row.refusal, reason: row.refusal.reason(), cause: failure, level: row.level}
}

func refusalCausing(rows []causedRefusal, failure error) (causedRefusal, bool) {
	matching := slices.IndexFunc(rows, func(row causedRefusal) bool { return errors.Is(failure, row.cause) })
	if matching < 0 {
		return causedRefusal{}, false
	}
	return rows[matching], true
}
