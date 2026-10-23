// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func runRecordingFor(configured configuration.Configuration, evaluation evaluating) (application.RecordRun, error) {
	callers, err := actionsCallersFor(configured, evaluation.endpoint)
	if err != nil {
		return application.RecordRun{}, err
	}
	publisher, err := evaluation.app.repositoryCheckRunsFor(configured.InstallationIDs())
	if err != nil {
		return application.RecordRun{}, err
	}
	return application.RecordRun{Ledger: evaluation.ledger, Publisher: publisher, Callers: callers}, nil
}

type repositoryCheckRuns struct {
	app           appForge
	installations github.RepositoryInstallations
	allowed       []int64
}

var errInstallationNotAllowlisted = errors.New("the repository's App installation is not on the configured allowlist")

func (app appForge) repositoryCheckRunsFor(allowed []int64) (repositoryCheckRuns, error) {
	installations, err := github.ParseRepositoryInstallations(app.settings, app.key, app.clock)
	if err != nil {
		return repositoryCheckRuns{}, fmt.Errorf("build the repository installation lookup: %w", err)
	}
	return repositoryCheckRuns{app: app, installations: installations, allowed: allowed}, nil
}

func (publishing repositoryCheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	installation, err := publishing.installations.InstallationOf(ctx, locator.GovernedRepository())
	if err != nil {
		return err
	}
	if !slices.Contains(publishing.allowed, installation) {
		return fmt.Errorf("%w: installation %d holds %s", errInstallationNotAllowlisted, installation, locator.GovernedRepository())
	}
	installed, err := publishing.app.installationForgeFor(installation)
	if err != nil {
		return fmt.Errorf("build the forge of installation %d: %w", installation, err)
	}
	return installed.checkRuns.PublishCheckRun(ctx, locator, head, run)
}
