// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var windowOpens = time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)

func TestUploadRateRefusesTheUploadOneOverTheLimit(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	rate := application.NewUploadRate(limits, func() time.Time { return windowOpens })
	repository := parsedRepository(t)
	admitAll(t, rate, repository, limits.MaxUploadsPerMinute)

	err := rate.Admit(repository)

	if !errors.Is(err, application.ErrUploadRateExceeded) {
		t.Fatalf("want ErrUploadRateExceeded, got %v", err)
	}
}

func TestUploadRateAdmitsAgainOnceTheEarliestUploadIsAMinuteOld(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	clock := windowOpens
	rate := application.NewUploadRate(limits, func() time.Time { return clock })
	repository := parsedRepository(t)
	admitAll(t, rate, repository, limits.MaxUploadsPerMinute)

	clock = windowOpens.Add(time.Minute)

	if err := rate.Admit(repository); err != nil {
		t.Fatalf("an upload a minute after the first was refused: %v", err)
	}
}

func TestUploadRateSaysHowLongUntilTheEarliestUploadLeavesTheWindow(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	clock := windowOpens
	rate := application.NewUploadRate(limits, func() time.Time { return clock })
	repository := parsedRepository(t)
	admitAll(t, rate, repository, limits.MaxUploadsPerMinute)
	clock = windowOpens.Add(20*time.Second + 500*time.Millisecond)

	var refusal application.UploadRateRefusal
	if err := rate.Admit(repository); !errors.As(err, &refusal) {
		t.Fatalf("want an UploadRateRefusal, got %v", err)
	}

	if refusal.RetryAfter != 40*time.Second {
		t.Fatalf("retry after %s, want 40s: 39.5s left, rounded up to whole seconds", refusal.RetryAfter)
	}
}

func TestUploadRateAdmitsTheUploadAtTheLimit(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	rate := application.NewUploadRate(limits, func() time.Time { return windowOpens })
	repository := parsedRepository(t)
	admitAll(t, rate, repository, limits.MaxUploadsPerMinute-1)

	if err := rate.Admit(repository); err != nil {
		t.Fatalf("the upload at the limit was refused: %v", err)
	}
}

func TestUploadRateStillRefusesJustShortOfAMinute(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	clock := windowOpens
	rate := application.NewUploadRate(limits, func() time.Time { return clock })
	repository := parsedRepository(t)
	admitAll(t, rate, repository, limits.MaxUploadsPerMinute)

	clock = windowOpens.Add(time.Minute - time.Nanosecond)

	if err := rate.Admit(repository); !errors.Is(err, application.ErrUploadRateExceeded) {
		t.Fatalf("want ErrUploadRateExceeded a nanosecond short of a minute, got %v", err)
	}
}

func TestUploadRateNeverSpendsOneRepositorysAllowanceOnAnother(t *testing.T) {
	limits := domain.DeclaredCoverageLimits()
	rate := application.NewUploadRate(limits, func() time.Time { return windowOpens })
	admitAll(t, rate, parsedStranger(t), limits.MaxUploadsPerMinute)

	if err := rate.Admit(parsedRepository(t)); err != nil {
		t.Fatalf("another repository's uploads spent this one's allowance: %v", err)
	}
}

func admitAll(t *testing.T, rate *application.UploadRate, repository domain.GovernedRepository, uploads int) {
	t.Helper()

	for upload := range uploads {
		if err := rate.Admit(repository); err != nil {
			t.Fatalf("upload %d of %d was refused: %v", upload+1, uploads, err)
		}
	}
}
