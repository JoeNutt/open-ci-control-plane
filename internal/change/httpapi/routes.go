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
	DecisionIDs   domain.DecisionIDSource
}

func ParseRoutes(settings Settings) (http.Handler, error) {
	answering := answers{log: settings.Log}
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{
			allowed:   allowlist(slices.Clone(settings.Installations)),
			answering: answering,
			log:       settings.Log,
			gate:      settings.Gate,
			ids:       settings.DecisionIDs,
		},
		Secret:   settings.Secret,
		Refusals: refusals{log: settings.Log},
	})
	if err != nil {
		return nil, err
	}
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{answering: answering, check: check})
	return routes, nil
}

type events struct {
	answering answers
	check     ingest.SignatureCheck
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, refused := deliveryIn(asked.Header)
	if refused != nil {
		receiving.answering.write(asked.Context(), answer, refused)
		return
	}
	if refused := lengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.answering.write(asked.Context(), answer, refused)
		return
	}
	receiving.check.ServeHTTP(answer, withDelivery(asked, received))
}

type admitted struct {
	allowed   allowlist
	answering answers
	log       *slog.Logger
	gate      ChangeGate
	ids       domain.DecisionIDSource
}

func (stage admitted) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	stated, err := payloadOf(asked.Body)
	if err != nil {
		stage.answering.write(asked.Context(), answer, malformedPayloadProblem())
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
	stage.answering.write(asked.Context(), answer, stage.evaluatedAnswer(asked.Context(), received, stated))
}

func (stage admitted) evaluatedAnswer(ctx context.Context, received delivery, stated payload) answerBody {
	locator, err := stated.pullRequest()
	if err != nil {
		return unparseablePullRequestProblem()
	}
	id, _ := stage.ids.NextDecisionID()
	if err := stage.gate.Execute(ctx, locator, id); err != nil {
		stage.log.LogAttrs(ctx, slog.LevelError, failedEvaluationLine, slog.String(deliveryIDKey, received.id),
			slog.String(decisionIDKey, id.String()), slog.String(reasonKey, err.Error()))
		return failedEvaluationProblem()
	}
	return acceptanceOf(received, id)
}
