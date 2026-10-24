// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"slices"
)

const (
	compressedBytesLabel   = "compressed artifact bytes"
	decompressedBytesLabel = "decompressed bytes"
	fileEntriesLabel       = "file entries"
	suitesLabel            = "suites per repository"
	testSectionsLabel      = "TN sections"
	lineRecordsLabel       = "DA records"
	indexedTestsLabel      = "indexed tests"
	indexEdgesLabel        = "block-test edges"
)

var ErrCoverageBoundExceeded = errors.New("coverage artifact exceeds a bound")

var ErrCompressedArtifactTooLarge = boundNamed(compressedBytesLabel)

var ErrDecompressedArtifactTooLarge = boundNamed(decompressedBytesLabel)

var ErrTooManyFileEntries = boundNamed(fileEntriesLabel)

var ErrTooManySuites = boundNamed(suitesLabel)

var ErrTooManyTestSections = boundNamed(testSectionsLabel)

var ErrTooManyLineRecords = boundNamed(lineRecordsLabel)

var ErrTooManyIndexedTests = boundNamed(indexedTestsLabel)

var ErrTooManyIndexEdges = boundNamed(indexEdgesLabel)

type CoverageLimits struct {
	MaxCompressedBytes     int64
	MaxDecompressedBytes   int64
	MaxFileEntries         int
	MaxTestSections        int
	MaxLineRecords         int
	MaxSuitesPerRepository int
	MaxUploadsPerMinute    int
	MaxIndexedTests        int
	MaxIndexEdges          int
}

type coverageBound struct {
	exceeded  error
	label     string
	allowedIn func(CoverageLimits) int64
}

var (
	compressedBytesBound = coverageBound{ErrCompressedArtifactTooLarge, compressedBytesLabel,
		func(limits CoverageLimits) int64 { return limits.MaxCompressedBytes }}
	decompressedBytesBound = coverageBound{ErrDecompressedArtifactTooLarge, decompressedBytesLabel,
		func(limits CoverageLimits) int64 { return limits.MaxDecompressedBytes }}
	fileEntriesBound = coverageBound{ErrTooManyFileEntries, fileEntriesLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxFileEntries) }}
	suitesBound = coverageBound{ErrTooManySuites, suitesLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxSuitesPerRepository) }}
	testSectionsBound = coverageBound{ErrTooManyTestSections, testSectionsLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxTestSections) }}
	lineRecordsBound = coverageBound{ErrTooManyLineRecords, lineRecordsLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxLineRecords) }}
	indexedTestsBound = coverageBound{ErrTooManyIndexedTests, indexedTestsLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxIndexedTests) }}
	indexEdgesBound = coverageBound{ErrTooManyIndexEdges, indexEdgesLabel,
		func(limits CoverageLimits) int64 { return int64(limits.MaxIndexEdges) }}
)

var coverageBounds = []coverageBound{
	compressedBytesBound, decompressedBytesBound, fileEntriesBound, suitesBound, testSectionsBound, lineRecordsBound,
	indexedTestsBound, indexEdgesBound,
}

func DeclaredCoverageLimits() CoverageLimits {
	return CoverageLimits{
		MaxCompressedBytes:     64 << 20,
		MaxDecompressedBytes:   256 << 20,
		MaxFileEntries:         200_000,
		MaxTestSections:        10_000,
		MaxLineRecords:         2_000_000,
		MaxSuitesPerRepository: 5_000,
		MaxUploadsPerMinute:    60,
		MaxIndexedTests:        10_000,
		MaxIndexEdges:          5_000_000,
	}
}

func (limits CoverageLimits) FileEntriesBreach(fileEntries int) error {
	return fileEntriesBound.breachIn(limits, int64(fileEntries))
}

func (limits CoverageLimits) TestSectionsBreach(testSections int) error {
	return testSectionsBound.breachIn(limits, int64(testSections))
}

func (limits CoverageLimits) LineRecordsBreach(lineRecords int) error {
	return lineRecordsBound.breachIn(limits, int64(lineRecords))
}

func (limits CoverageLimits) indexedTestsBreach(indexedTests int) error {
	return indexedTestsBound.breachIn(limits, int64(indexedTests))
}

func (limits CoverageLimits) indexEdgesBreach(indexEdges int) error {
	return indexEdgesBound.breachIn(limits, int64(indexEdges))
}

func (limits CoverageLimits) SuitesBreach(suites int) error {
	return suitesBound.breachIn(limits, int64(suites))
}

func (limits CoverageLimits) CompressedBytesBreach(compressedBytes int64) error {
	return compressedBytesBound.breachIn(limits, compressedBytes)
}

func (limits CoverageLimits) DecompressedBytesBreach(decompressedBytes int64) error {
	return decompressedBytesBound.breachIn(limits, decompressedBytes)
}

func (limits CoverageLimits) LimitOf(bound error) (string, bool) {
	stated := slices.IndexFunc(coverageBounds, func(each coverageBound) bool { return each.exceeded == bound })
	if stated < 0 {
		return "", false
	}
	return coverageBounds[stated].limitIn(limits), true
}

func (bound coverageBound) breachIn(limits CoverageLimits, measured int64) error {
	allowed := bound.allowedIn(limits)
	if measured > allowed {
		return fmt.Errorf("%w: %d exceeds %d", bound.exceeded, measured, allowed)
	}
	return nil
}

func (bound coverageBound) limitIn(limits CoverageLimits) string {
	return fmt.Sprintf("%d %s", bound.allowedIn(limits), bound.label)
}

func boundNamed(label string) error {
	return fmt.Errorf("%w: %s", ErrCoverageBoundExceeded, label)
}
