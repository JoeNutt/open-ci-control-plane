// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"log/slog"
	"net/http"
)

type refusals struct{ log *slog.Logger }

func (recording refusals) RecordRefusal(refused *http.Request, reason error) {
	received, err := deliveryOf(refused.Context())
	if err != nil {
		recording.log.LogAttrs(refused.Context(), slog.LevelError, refusedDelivery,
			slog.String(reasonKey, reason.Error()), slog.String(faultKey, err.Error()))
		return
	}
	recording.log.LogAttrs(refused.Context(), slog.LevelWarn, refusedDelivery,
		slog.String(deliveryIDKey, received.id), slog.String(reasonKey, reason.Error()))
}

func (recording refusals) record(ctx context.Context, received delivery, reason string) {
	recording.log.LogAttrs(ctx, slog.LevelWarn, refusedDelivery,
		slog.String(deliveryIDKey, received.id), slog.String(reasonKey, reason))
}

func (recording refusals) recordUnidentified(ctx context.Context, refused fieldProblem) {
	recording.log.LogAttrs(ctx, slog.LevelWarn, refusedDelivery,
		slog.String(reasonKey, refused.reason()), slog.String(fieldKey, refused.Field))
}

func (recording refusals) logged(ctx context.Context, received delivery, refused refusal) refusal {
	recording.record(ctx, received, refused.reason())
	return refused
}
