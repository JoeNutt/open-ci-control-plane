// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"sync"
	"time"
)

type heldAcceptance struct {
	answered acceptance
	heldAt   time.Time
}

type acceptedDeliveries struct {
	mutex sync.Mutex
	byID  map[string]heldAcceptance
	clock func() time.Time
}

func newAcceptedDeliveries(clock func() time.Time) *acceptedDeliveries {
	return &acceptedDeliveries{byID: map[string]heldAcceptance{}, clock: clock}
}

func (accepted *acceptedDeliveries) originalOf(received delivery) (acceptance, bool) {
	accepted.mutex.Lock()
	defer accepted.mutex.Unlock()
	original, isHeld := accepted.byID[received.id]
	if isHeld && !accepted.clock().Before(original.heldAt.Add(24*time.Hour)) {
		return acceptance{}, false
	}
	return original.answered, isHeld
}

func (accepted *acceptedDeliveries) held(answered acceptance) acceptance {
	accepted.mutex.Lock()
	defer accepted.mutex.Unlock()
	accepted.byID[answered.DeliveryID] = heldAcceptance{answered: answered, heldAt: accepted.clock()}
	return answered
}
