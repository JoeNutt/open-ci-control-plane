// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type answerBody interface {
	status() int
}

type headedAnswer interface {
	writeHeaders(headers http.Header)
}

type refusal interface {
	answerBody
	reason() string
}

type answers struct{ log *slog.Logger }

func (writing answers) write(ctx context.Context, answer http.ResponseWriter, body answerBody) {
	answer.Header().Set(contentTypeHeader, jsonMediaType)
	if headed, isHeaded := body.(headedAnswer); isHeaded {
		headed.writeHeaders(answer.Header())
	}
	answer.WriteHeader(body.status())
	if err := json.NewEncoder(answer).Encode(body); err != nil {
		writing.log.LogAttrs(ctx, slog.LevelWarn, unwrittenAnswer, slog.String("error", err.Error()))
	}
}
