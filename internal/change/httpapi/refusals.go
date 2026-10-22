// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"log/slog"
	"net/http"
)

const (
	refusedDelivery  = "delivery refused"
	uncarriedRefusal = "a delivery was refused without the headers the listener parsed"
	deliveryIDKey    = "deliveryId"
	reasonKey        = "reason"
)

type refusals struct{ log *slog.Logger }

func (recording refusals) RecordRefusal(refused *http.Request, reason error) {
	received, isCarried := deliveryOf(refused.Context())
	if !isCarried {
		recording.log.LogAttrs(refused.Context(), slog.LevelError, uncarriedRefusal,
			slog.String(reasonKey, reason.Error()))
		return
	}
	recording.log.LogAttrs(refused.Context(), slog.LevelWarn, refusedDelivery,
		slog.String(deliveryIDKey, received.id), slog.String(reasonKey, reason.Error()))
}
