// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"errors"
	"slices"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const uploadRateWindow = time.Minute

var ErrUploadRateExceeded = errors.New("coverage uploads exceed the repository's rate")

type UploadRate struct {
	limits   domain.CoverageLimits
	now      func() time.Time
	admitted map[domain.GovernedRepository][]time.Time
}

func NewUploadRate(limits domain.CoverageLimits, now func() time.Time) *UploadRate {
	return &UploadRate{limits: limits, now: now, admitted: map[domain.GovernedRepository][]time.Time{}}
}

func (rate *UploadRate) Admit(repository domain.GovernedRepository) error {
	now := rate.now()
	recent := rate.admissionsWithinWindow(repository, now)
	if len(recent) >= rate.limits.MaxUploadsPerMinute {
		left := recent[0].Add(uploadRateWindow).Sub(now)
		return UploadRateRefusal{RetryAfter: (left + time.Second - 1).Truncate(time.Second)}
	}
	rate.admitted[repository] = append(recent, now)
	return nil
}

func (rate *UploadRate) admissionsWithinWindow(repository domain.GovernedRepository, now time.Time) []time.Time {
	admitted := rate.admitted[repository]
	firstRecent := slices.IndexFunc(admitted, func(at time.Time) bool { return now.Sub(at) < uploadRateWindow })
	if firstRecent < 0 {
		return nil
	}
	return admitted[firstRecent:]
}

type UploadRateRefusal struct {
	RetryAfter time.Duration
}

func (refusal UploadRateRefusal) Error() string { return ErrUploadRateExceeded.Error() }

func (refusal UploadRateRefusal) Is(target error) bool { return target == ErrUploadRateExceeded }
