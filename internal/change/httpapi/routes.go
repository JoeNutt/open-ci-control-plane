// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

type ChangeGate interface {
	Execute(ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID) error
}

type Settings struct {
	Log           *slog.Logger
	Secret        domain.WebhookSecret
	Installations []int64
	Gate          ChangeGate
}

func ParseRoutes(settings Settings) (http.Handler, error) {
	refusing := problems{log: settings.Log}
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{allowed: allowlist(slices.Clone(settings.Installations)), refusing: refusing, gate: settings.Gate},
		Secret:   settings.Secret,
		Refusals: refusals{log: settings.Log},
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
	gate     ChangeGate
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
	received, _ := deliveryOf(asked.Context())
	if !isEvaluated(received.event, stated.Action) {
		answer.WriteHeader(http.StatusNoContent)
		return
	}
	stage.evaluate(answer, asked, stated)
}

func (stage admitted) evaluate(answer http.ResponseWriter, asked *http.Request, stated payload) {
	locator, err := stated.pullRequest()
	if err != nil {
		stage.refusing.refuse(asked.Context(), answer, unparseablePullRequestProblem())
		return
	}
	_ = stage.gate.Execute(asked.Context(), locator, domain.DecisionID{})
	answer.WriteHeader(http.StatusAccepted)
}
