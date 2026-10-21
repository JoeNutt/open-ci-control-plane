// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
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

func TestTheOldestDeliveryGivesWayWhenTheBoundIsReached(t *testing.T) {
	now := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	accepted := newAcceptedDeliveries(func() time.Time { return now })
	for held := range maxHeldDeliveries + 1 {
		now = now.Add(time.Second)
		accepted.held(acceptance{DeliveryID: fmt.Sprintf("delivery-%d", held)})
	}

	if len(accepted.byID) != maxHeldDeliveries {
		t.Errorf("%d deliveries are held, want the bound, %d", len(accepted.byID), maxHeldDeliveries)
	}
	if _, isHeld := accepted.originalOf(delivery{id: "delivery-0"}); isHeld {
		t.Error("the oldest delivery is still held past the bound")
	}
	if _, isHeld := accepted.originalOf(delivery{id: fmt.Sprintf("delivery-%d", maxHeldDeliveries)}); !isHeld {
		t.Error("the newest delivery was not held")
	}
}
