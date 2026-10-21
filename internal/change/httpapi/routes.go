// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func ParseRoutes(log *slog.Logger, secret domain.WebhookSecret, installations []int64) (http.Handler, error) {
	refusing := problems{log: log}
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{allowed: allowlist(slices.Clone(installations)), refusing: refusing},
		Secret:   secret,
		Refusals: refusals{log: log},
	})
	if err != nil {
		return nil, err
	}
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{refusing: refusing, check: check})
	return routes, nil
}

type events struct {
	refusing problems
	check    ingest.SignatureCheck
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, refused := deliveryIn(asked.Header)
	if refused != nil {
		receiving.refusing.refuse(asked.Context(), answer, refused)
		return
	}
	if refused := lengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.refusing.refuse(asked.Context(), answer, refused)
		return
	}
	receiving.check.ServeHTTP(answer, withDelivery(asked, received))
}

type admitted struct {
	allowed  allowlist
	refusing problems
}

func (stage admitted) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	stated, err := payloadOf(asked.Body)
	if err != nil {
		stage.refusing.refuse(asked.Context(), answer, malformedPayloadProblem())
		return
	}
	if !stage.allowed.admits(stated) {
		answer.WriteHeader(http.StatusNotFound)
		return
	}
	answer.WriteHeader(http.StatusNoContent)
}
