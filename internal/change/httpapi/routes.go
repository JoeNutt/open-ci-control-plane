// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const eventsRoute = "POST /v1/forge/github/events"

func Routes(log *slog.Logger) http.Handler {
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{log: log})
	return routes
}

type events struct {
	log *slog.Logger
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	if refusal, isRefused := headerProblemIn(asked.Header); isRefused {
		receiving.refuse(asked.Context(), answer, refusal)
		return
	}
	if asked.ContentLength > ingest.MaxDeliveryBytes {
		answer.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
}

func (receiving events) refuse(ctx context.Context, answer http.ResponseWriter, refusal fieldProblem) {
	answer.Header().Set(contentTypeHeader, jsonMediaType)
	answer.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(answer).Encode(refusal); err != nil {
		receiving.log.LogAttrs(ctx, slog.LevelWarn, unwrittenAnswer, slog.String("error", err.Error()))
	}
}
