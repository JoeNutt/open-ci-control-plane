// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"testing"
	"time"
)

func TestLapsedDeliveriesAreDroppedWhenAnotherIsHeld(t *testing.T) {
	now := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	accepted := newAcceptedDeliveries(func() time.Time { return now })
	accepted.held(acceptance{DeliveryID: "first"})
	accepted.held(acceptance{DeliveryID: "second"})

	now = now.Add(replayWindow)
	accepted.held(acceptance{DeliveryID: "third"})

	if len(accepted.byID) != 1 {
		t.Errorf("%d deliveries are held, want only the one inside the window", len(accepted.byID))
	}
}
