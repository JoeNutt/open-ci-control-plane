// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func ParseRoutes(settings Settings) (http.Handler, error) {
	if missing := slices.IndexFunc(requiredParts, isMissingFrom(settings)); missing >= 0 {
		return nil, fmt.Errorf("%w: %s", ErrIncompleteRoutes, requiredParts[missing].name)
	}
	answering := answers{log: settings.Log}
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{
			allowed:   allowlist(slices.Clone(settings.Installations)),
			answering: answering,
			log:       settings.Log,
			gate:      settings.Gate,
			ids:       settings.DecisionIDs,
			refusing:  refusals{log: settings.Log},
		},
		Secret:   settings.Secret,
		Refusals: refusals{log: settings.Log},
	})
	if err != nil {
		return nil, fmt.Errorf("build the listener's signature check: %w", err)
	}
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{answering: answering, check: check, refusing: refusals{log: settings.Log}})
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
		receiving.answering.write(asked.Context(), answer, refused)
		return
	}
	if refused := lengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.refusing.record(asked.Context(), received, "the declared length")
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
	refusing  refusals
}

func (stage admitted) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, err := deliveryOf(asked.Context())
	if err != nil {
		stage.answering.write(asked.Context(), answer, stage.failedEvaluationAnswer(asked.Context(), err))
		return
	}
	stated, err := payloadOf(asked.Body)
	if err != nil {
		stage.refusing.record(asked.Context(), received, "malformed payload")
		stage.answering.write(asked.Context(), answer, malformedPayloadProblem())
		return
	}
	if !stage.allowed.admits(stated) {
		stage.refusing.record(asked.Context(), received, "installation not allowlisted")
		answer.WriteHeader(http.StatusNotFound)
		return
	}
	if !isEvaluated(received.event, stated.Action) {
		answer.WriteHeader(http.StatusNoContent)
		return
	}
	stage.answering.write(asked.Context(), answer, stage.evaluatedAnswer(asked.Context(), received, stated))
}

func (stage admitted) evaluatedAnswer(ctx context.Context, received delivery, stated payload) answerBody {
	locator, err := stated.pullRequest()
	if err != nil {
		stage.refusing.record(ctx, received, "unparseable pull request")
		return unparseablePullRequestProblem()
	}
	id, err := stage.ids.NextDecisionID()
	if err != nil {
		return stage.failedEvaluationAnswer(ctx, err, slog.String(deliveryIDKey, received.id))
	}
	if err := stage.gate.Execute(ctx, locator, id); err != nil {
		return stage.failedEvaluationAnswer(ctx, err,
			slog.String(deliveryIDKey, received.id), slog.String(decisionIDKey, id.String()))
	}
	return acceptanceOf(received, id)
}

func (stage admitted) failedEvaluationAnswer(ctx context.Context, cause error, named ...slog.Attr) answerBody {
	stage.log.LogAttrs(ctx, slog.LevelError, failedEvaluationLine,
		append(named, slog.String(reasonKey, cause.Error()))...)
	return failedEvaluationProblem()
}
