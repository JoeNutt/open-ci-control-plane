// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"slices"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

type repositoryCheckRuns struct {
	app           appForge
	installations github.RepositoryInstallations
	allowed       []int64
}

func (app appForge) repositoryCheckRunsFor(allowed []int64) repositoryCheckRuns {
	installations, _ := github.ParseRepositoryInstallations(app.settings, app.key, app.clock)
	return repositoryCheckRuns{app: app, installations: installations, allowed: allowed}
}

func (publishing repositoryCheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, head domain.CommitSHA, run domain.CheckRun,
) error {
	installation, err := publishing.installations.InstallationOf(ctx, locator.GovernedRepository())
	if err != nil {
		return err
	}
	if !slices.Contains(publishing.allowed, installation) {
		return errors.New("not allowlisted")
	}
	installed, err := publishing.app.installationForgeFor(installation)
	if err != nil {
		return err
	}
	return installed.checkRuns.PublishCheckRun(ctx, locator, head, run)
}
