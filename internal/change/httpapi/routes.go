// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
)

const eventsRoute = "POST /v1/forge/github/events"

var requiredHeaders = []string{eventHeader, deliveryHeader}

func Routes(log *slog.Logger) http.Handler {
	routes := http.NewServeMux()
	routes.Handle(eventsRoute, events{log: log})
	return routes
}

type events struct {
	log *slog.Logger
}

func (receiving events) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	if absent := slices.IndexFunc(requiredHeaders, isAbsentFrom(asked.Header)); absent >= 0 {
		receiving.refuse(asked.Context(), answer, missingHeaderProblem(requiredHeaders[absent]))
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

func isAbsentFrom(stated http.Header) func(string) bool {
	return func(name string) bool { return stated.Get(name) == "" }
}
