// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
)

var windowOpens = time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)

func TestUploadRateRefusesTheUploadOneOverTheLimit(t *testing.T) {
	rate := application.NewUploadRate(func() time.Time { return windowOpens })
	repository := parsedRepository(t)
	for upload := range 60 {
		if err := rate.Admit(repository); err != nil {
			t.Fatalf("upload %d was refused: %v", upload+1, err)
		}
	}

	err := rate.Admit(repository)

	if !errors.Is(err, application.ErrUploadRateExceeded) {
		t.Fatalf("want ErrUploadRateExceeded, got %v", err)
	}
}
