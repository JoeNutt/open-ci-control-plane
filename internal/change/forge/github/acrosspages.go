// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"errors"
	"fmt"
)

var ErrTooManyPages = errors.New("the forge offered more pages than the bound allows")

type paging struct {
	endpointAt func(page int) string
	perPage    int
	maxPages   int
}

func acrossPages[Item any](
	ctx context.Context, forge connection, over paging,
) ([]Item, error) {
	var collected []Item
	for page := 1; page <= over.maxPages; page++ {
		onPage, err := jsonAt[[]Item](ctx, forge, over.endpointAt(page))
		if err != nil {
			return nil, err
		}
		collected = append(collected, onPage...)
		if len(onPage) < over.perPage {
			return collected, nil
		}
	}
	return nil, fmt.Errorf("%w: %d pages of %d",
		ErrTooManyPages, over.maxPages, over.perPage)
}
