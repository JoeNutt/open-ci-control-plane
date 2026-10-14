// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"runtime"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestCheckRunTextFitsAtTheExactBoundaryAndNotOneShort(t *testing.T) {
	run := checkRunTextFixture(t)
	block := checkRunTextBlockOf(run.Violations[0])

	listedAt := map[string]struct{ capacity, listed int }{
		"exactly the block":   {len(block), 1},
		"one short the block": {len(block) - 1, 0},
	}

	for form, want := range listedAt {
		t.Run(form, func(t *testing.T) {
			assertCheckRunTextLists(t, run, want.capacity, want.listed)
		})
	}
}

func TestCheckRunTextCountsTheTruncationLineItWouldLeave(t *testing.T) {
	run := checkRunTextWithViolations(t, 10)
	block := checkRunTextBlockOf(run.Violations[0])
	leaves := checkRunTextTruncationOf(9, 10)

	listedAt := map[string]struct{ capacity, listed int }{
		"block and its line exactly":   {len(block) + len(leaves), 1},
		"block and its line one short": {len(block) + len(leaves) - 1, 0},
	}

	for form, want := range listedAt {
		t.Run(form, func(t *testing.T) {
			assertCheckRunTextLists(t, run, want.capacity, want.listed)
		})
	}
}

func TestCheckRunTextStopsAtTheFirstViolationThatWillNotFit(t *testing.T) {
	run := checkRunTextWithViolations(t, 2)
	run.Violations[0].Message = strings.Repeat("m", 500)
	small := checkRunTextBlockOf(run.Violations[1])

	assertCheckRunTextLists(t, run, len(small)+200, 0)
}

func TestCheckRunTextLeastLengthNeverStopsABlockThatFits(t *testing.T) {
	run := checkRunTextFixture(t)
	run.Violations[0].Message = strings.Repeat("a\r\n", 300)

	assertCheckRunTextLists(t, run, len(checkRunTextBlockOf(run.Violations[0])), 1)
}

func TestCheckRunTextDoesNotFormatAFieldThatCannotFit(t *testing.T) {
	const fieldBytes = 4 << 20
	run := checkRunTextFixture(t)
	run.Violations[0].Message = strings.Repeat("m", fieldBytes)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	output, err := checkRunTextFor(run, checkRunTextTestCapacity)
	runtime.ReadMemStats(&after)

	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(output.text, "**Rule**") {
		t.Fatalf("a %d-byte message was listed in a capacity of %d", fieldBytes, checkRunTextTestCapacity)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > fieldBytes/4 {
		t.Errorf("rendering allocated %d bytes for a field it could never list", allocated)
	}
}

func checkRunTextWithViolations(t *testing.T, count int) domain.CheckRun {
	t.Helper()

	run := checkRunTextFixture(t)
	template := run.Violations[0]
	run.Violations = nil
	for range count {
		run.Violations = append(run.Violations, template)
	}
	run.ViolationCount = count
	return run
}

func assertCheckRunTextLists(t *testing.T, run domain.CheckRun, capacity, listed int) {
	t.Helper()

	output, err := checkRunTextFor(run, capacity)
	if err != nil {
		t.Fatalf("render at capacity %d: %v", capacity, err)
	}
	if got := strings.Count(output.text, "**Rule**"); got != listed {
		t.Errorf("at capacity %d listed %d, want %d:\n%s", capacity, got, listed, output.text)
	}
	if len(output.text) > capacity {
		t.Errorf("text is %d bytes, capacity %d", len(output.text), capacity)
	}
}
