// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func ParseRoutes(settings Settings) (http.Handler, error) {
	if missing := slices.IndexFunc(requiredParts, isMissingFrom(settings)); missing >= 0 {
		return nil, fmt.Errorf("%w: %s", ErrIncompleteRoutes, requiredParts[missing].name)
	}
	answering := answers{log: settings.Log}
	refusing := refusals{log: settings.Log}
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{
			allowed:   allowlist(slices.Clone(settings.Installations)),
			answering: answering,
			refusing:  refusing,
			evaluating: evaluation{
				gate:     settings.Gate,
				ids:      settings.DecisionIDs,
				log:      settings.Log,
				refusing: refusing,
			},
		},
		Secret:   settings.Secret,
		Refusals: refusing,
	})
	if err != nil {
		return nil, fmt.Errorf("build the listener's signature check: %w", err)
	}
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{answering: answering, check: check, refusing: refusing})
	return routes, nil
}

type events struct {
	answering answers
	check     ingest.SignatureCheck
	refusing  refusals
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, refused := deliveryIn(asked.Header)
	if refused != nil {
		problem, _ := refused.(fieldProblem)
		receiving.refusing.log.LogAttrs(asked.Context(), slog.LevelWarn, refusedDelivery,
			slog.String(reasonKey, problem.Code), slog.String("field", problem.Field))
		receiving.answering.write(asked.Context(), answer, refused)
		return
	}
	if refused := lengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.answering.write(asked.Context(), answer, receiving.refusing.logged(asked.Context(), received, refused))
		return
	}
	receiving.check.ServeHTTP(answer, withDelivery(asked, received))
}

type admitted struct {
	allowed    allowlist
	answering  answers
	refusing   refusals
	evaluating evaluation
}

func (stage admitted) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, err := deliveryOf(asked.Context())
	if err != nil {
		stage.answering.write(asked.Context(), answer, stage.evaluating.failedAnswer(asked.Context(), err))
		return
	}
	stated, err := payloadOf(asked.Body)
	if err != nil {
		stage.answering.write(asked.Context(), answer,
			stage.refusing.logged(asked.Context(), received, malformedPayloadProblem()))
		return
	}
	if !stage.allowed.admits(stated) {
		stage.refusing.record(asked.Context(), received, notAllowlistedReason)
		answer.WriteHeader(http.StatusNotFound)
		return
	}
	if !isEvaluated(received.event, stated.Action) {
		answer.WriteHeader(http.StatusNoContent)
		return
	}
	stage.answering.write(asked.Context(), answer, stage.evaluating.answerFor(asked.Context(), received, stated))
}
