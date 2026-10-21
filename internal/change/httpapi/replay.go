// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"sync"
	"time"
)

const replayWindow = 24 * time.Hour

type heldAcceptance struct {
	answered acceptance
	heldAt   time.Time
}

func (held heldAcceptance) isLapsedAt(now time.Time) bool {
	return !now.Before(held.heldAt.Add(replayWindow))
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
	if !isHeld || original.isLapsedAt(accepted.clock()) {
		return acceptance{}, false
	}
	return original.answered, true
}

func (accepted *acceptedDeliveries) held(answered acceptance) acceptance {
	accepted.mutex.Lock()
	defer accepted.mutex.Unlock()
	accepted.byID[answered.DeliveryID] = heldAcceptance{answered: answered, heldAt: accepted.clock()}
	return answered
}
