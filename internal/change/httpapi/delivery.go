// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"net/http"
)

type delivery struct {
	event string
	id    string
}

type deliveryKey struct{}

func withDelivery(asked *http.Request, received delivery) *http.Request {
	return asked.WithContext(context.WithValue(asked.Context(), deliveryKey{}, received))
}

func deliveryOf(ctx context.Context) (delivery, bool) {
	received, isCarried := ctx.Value(deliveryKey{}).(delivery)
	return received, isCarried
}
