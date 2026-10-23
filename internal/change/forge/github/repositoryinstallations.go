// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type RepositoryInstallations struct {
	asApp connection
}

func ParseRepositoryInstallations(settings Settings, app Installation) (RepositoryInstallations, error) {
	issuer, err := parseAppJWTIssuer(settings, app)
	if err != nil {
		return RepositoryInstallations{}, err
	}
	asApp, err := parseConnection(settings, issuer)
	if err != nil {
		return RepositoryInstallations{}, err
	}
	return RepositoryInstallations{asApp: asApp}, nil
}

type installationPayload struct {
	ID int64 `json:"id"`
}

func (installations RepositoryInstallations) InstallationOf(ctx context.Context, repository domain.GovernedRepository) (int64, error) {
	installed, err := exchanged[installationPayload](ctx, installations.asApp, exchange{
		method:    http.MethodGet,
		endpoint:  repositoryEndpoint(repository) + "/installation",
		succeeded: http.StatusOK,
		notFound:  ErrNoSuchInstallation,
	})
	if err != nil {
		return 0, err
	}
	return installed.ID, nil
}
