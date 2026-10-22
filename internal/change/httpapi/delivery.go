// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"errors"
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

var errUncarriedDelivery = errors.New("the headers the listener parsed did not reach this stage")

func deliveryOf(ctx context.Context) (delivery, error) {
	received, isCarried := ctx.Value(deliveryKey{}).(delivery)
	if !isCarried {
		return delivery{}, errUncarriedDelivery
	}
	return received, nil
}
