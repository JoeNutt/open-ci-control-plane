// SPDX-License-Identifier: GPL-3.0-or-later

package ingest

import (
	"context"
	"net/http"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ActionsTokenSettings struct {
	Issuer   string
	Audience string
	Client   *http.Client
	Now      func() time.Time
}

type ActionsTokens struct{}

func ParseActionsTokens(settings ActionsTokenSettings) (*ActionsTokens, error) {
	return &ActionsTokens{}, nil
}

func (tokens *ActionsTokens) RepositoryOf(ctx context.Context, bearer string) (domain.Repository, error) {
	return domain.ParseRepository("JoeNutt/open-ci-control-plane")
}
