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
	admitted map[domain.GovernedRepository]int
}

func NewUploadRate(limits domain.CoverageLimits, now func() time.Time) *UploadRate {
	return &UploadRate{limits: limits, now: now, admitted: map[domain.GovernedRepository]int{}}
}

func (rate *UploadRate) Admit(repository domain.GovernedRepository) error {
	if rate.admitted[repository] >= rate.limits.MaxUploadsPerMinute {
		return ErrUploadRateExceeded
	}
	rate.admitted[repository]++
	return nil
}
