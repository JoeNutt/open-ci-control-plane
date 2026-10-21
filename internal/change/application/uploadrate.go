// SPDX-License-Identifier: GPL-3.0-or-later

package application

import (
	"errors"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrUploadRateExceeded = errors.New("coverage uploads exceed the repository's rate")

type UploadRate struct {
	now      func() time.Time
	admitted map[domain.GovernedRepository]int
}

func NewUploadRate(now func() time.Time) *UploadRate {
	return &UploadRate{now: now, admitted: map[domain.GovernedRepository]int{}}
}

func (rate *UploadRate) Admit(repository domain.GovernedRepository) error {
	if rate.admitted[repository] >= 60 {
		return ErrUploadRateExceeded
	}
	rate.admitted[repository]++
	return nil
}
