// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func ParseRoutes(log *slog.Logger, secret domain.WebhookSecret, installations []int64) (http.Handler, error) {
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted: admitted{allowed: allowlist(slices.Clone(installations))},
		Secret:   secret,
		Refusals: refusals{log: log},
	})
	if err != nil {
		return nil, err
	}
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{log: log, check: check})
	return routes, nil
}

type events struct {
	log   *slog.Logger
	check ingest.SignatureCheck
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	received, refused := deliveryIn(asked.Header)
	if refused != nil {
		receiving.refuse(asked.Context(), answer, refused)
		return
	}
	if refused := lengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.refuse(asked.Context(), answer, refused)
		return
	}
	receiving.check.ServeHTTP(answer, withDelivery(asked, received))
}

func (receiving events) refuse(ctx context.Context, answer http.ResponseWriter, refused refusal) {
	answer.Header().Set(contentTypeHeader, jsonMediaType)
	answer.WriteHeader(refused.status())
	if err := json.NewEncoder(answer).Encode(refused); err != nil {
		receiving.log.LogAttrs(ctx, slog.LevelWarn, unwrittenAnswer, slog.String("error", err.Error()))
	}
}

type admitted struct{ allowed allowlist }

func (stage admitted) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	stated, err := payloadOf(asked.Body)
	if err != nil {
		answer.Header().Set(contentTypeHeader, jsonMediaType)
		answer.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(answer).Encode(fieldProblem{Code: "malformed_payload", Message: "the signed body is not a JSON object", Field: "body"})
		return
	}
	if !stage.allowed.admits(stated) {
		answer.WriteHeader(http.StatusNotFound)
		return
	}
	answer.WriteHeader(http.StatusNoContent)
}
