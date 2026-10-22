// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/coverage/memory"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/ingest"
)

const (
	actionsTokenIssuer   = "https://token.actions.githubusercontent.com"
	maxTreeResponseBytes = 8 << 20
)

func coverageReplacementFor(
	configured configuration.Configuration, app appForge, endpoint forgeEndpoint,
) (application.ReplaceCoverage, error) {
	issuer, err := domain.ParseHTTPSOrigin(actionsTokenIssuer)
	if err != nil {
		return application.ReplaceCoverage{}, err
	}
	callers, err := ingest.ParseActionsTokens(ingest.ActionsTokenSettings{
		Issuer: issuer, Audience: configured.InstanceURL(), Client: forgeClientTrusting(endpoint.trusted), Now: time.Now,
	})
	if err != nil {
		return application.ReplaceCoverage{}, err
	}
	paths, err := app.repositoryPathsFor(configured.InstallationIDs())
	if err != nil {
		return application.ReplaceCoverage{}, err
	}
	limits := domain.DeclaredCoverageLimits()
	rate, err := application.ParseUploadRate(limits, time.Now)
	if err != nil {
		return application.ReplaceCoverage{}, err
	}
	return application.ReplaceCoverage{
		Maps: memory.NewCoverageMaps(limits), Paths: paths, Callers: callers, Rate: rate,
	}, nil
}

func (app appForge) repositoryPathsFor(installations []int64) (installationsRepositoryPaths, error) {
	each := make(installationsRepositoryPaths, 0, len(installations))
	for _, installation := range installations {
		settings, err := app.settingsFor(installation)
		if err != nil {
			return nil, err
		}
		settings.MaxResponseBytes = maxTreeResponseBytes
		paths, err := github.ParseRepositoryPaths(settings)
		if err != nil {
			return nil, refusedInstallation(installation, err)
		}
		each = append(each, paths)
	}
	return each, nil
}

type installationsRepositoryPaths []github.RepositoryPaths

func (each installationsRepositoryPaths) KnownPathsOf(
	ctx context.Context, repository domain.GovernedRepository,
) (domain.KnownPaths, error) {
	for _, paths := range each {
		known, err := paths.KnownPathsOf(ctx, repository)
		if !errors.Is(err, github.ErrNoSuchRepository) {
			return known, err
		}
	}
	return domain.KnownPaths{}, fmt.Errorf("%w: %s, under every configured installation", github.ErrNoSuchRepository, repository)
}
