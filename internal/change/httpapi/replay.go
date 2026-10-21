// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import "sync"

type acceptedDeliveries struct {
	mutex sync.Mutex
	byID  map[string]acceptance
}

func newAcceptedDeliveries() *acceptedDeliveries {
	return &acceptedDeliveries{byID: map[string]acceptance{}}
}

func (accepted *acceptedDeliveries) originalOf(received delivery) (acceptance, bool) {
	accepted.mutex.Lock()
	defer accepted.mutex.Unlock()
	original, isHeld := accepted.byID[received.id]
	return original, isHeld
}

func (accepted *acceptedDeliveries) held(answered acceptance) acceptance {
	accepted.mutex.Lock()
	defer accepted.mutex.Unlock()
	accepted.byID[answered.DeliveryID] = answered
	return answered
}
