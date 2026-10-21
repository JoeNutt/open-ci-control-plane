// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func ParseRoutes(log *slog.Logger, secret domain.WebhookSecret) (http.Handler, error) {
	check, err := ingest.ParseSignatureCheck(ingest.Settings{
		Accepted:   admitted{},
		Secret:     secret,
		Rejections: refusals{log: log},
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
	if refusal, isRefused := headerProblemIn(asked.Header); isRefused {
		receiving.refuse(asked.Context(), answer, refusal)
		return
	}
	if refusal, isRefused := lengthProblemIn(asked.ContentLength); isRefused {
		receiving.refuse(asked.Context(), answer, refusal)
		return
	}
	receiving.check.ServeHTTP(answer, asked)
}

func (receiving events) refuse(ctx context.Context, answer http.ResponseWriter, refused refusal) {
	answer.Header().Set(contentTypeHeader, jsonMediaType)
	answer.WriteHeader(refused.status())
	if err := json.NewEncoder(answer).Encode(refused); err != nil {
		receiving.log.LogAttrs(ctx, slog.LevelWarn, unwrittenAnswer, slog.String("error", err.Error()))
	}
}

type admitted struct{}

func (admitted) ServeHTTP(answer http.ResponseWriter, _ *http.Request) {
	answer.WriteHeader(http.StatusNoContent)
}

type refusals struct{ log *slog.Logger }

func (logging refusals) CountRejectedDelivery(refused *http.Request, reason error) {
	logging.log.LogAttrs(refused.Context(), slog.LevelWarn, "delivery refused",
		slog.String("deliveryId", refused.Header.Get(deliveryHeader)),
		slog.String("reason", reason.Error()))
}
