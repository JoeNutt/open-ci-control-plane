// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"errors"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

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
	var recent []time.Time
	for _, at := range rate.admitted[repository] {
		if now.Sub(at) < time.Minute {
			recent = append(recent, at)
		}
	}
	rate.admitted[repository] = recent
	if len(recent) >= rate.limits.MaxUploadsPerMinute {
		return ErrUploadRateExceeded
	}
	rate.admitted[repository] = append(recent, now)
	return nil
}
