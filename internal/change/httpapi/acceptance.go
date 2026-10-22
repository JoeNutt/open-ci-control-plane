// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type acceptance struct {
	DeliveryID string `json:"deliveryId"`
	DecisionID string `json:"decisionId"`
	Accepted   bool   `json:"accepted"`
}

func (acceptance) status() int { return http.StatusAccepted }

func acceptanceOf(received delivery, id domain.DecisionID) acceptance {
	return acceptance{DeliveryID: received.id, DecisionID: id.String(), Accepted: true}
}
