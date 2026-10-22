// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

type appForge struct {
	settings github.Settings
	key      github.AppKey
	clock    func() time.Time
}

type forgeEndpoint struct {
	baseURL string
	trusted *x509.CertPool
}

func appForgeOf(configured configuration.Configuration, endpoint forgeEndpoint) (appForge, error) {
	key, err := configuration.AppKeyParsedBy(configured, github.ParseAppKey)
	if err != nil {
		return appForge{}, err
	}
	return appForge{settings: github.Settings{
		BaseURL:          endpoint.baseURL,
		AppID:            configured.AppID(),
		Client:           forgeClientTrusting(endpoint.trusted),
		MaxResponseBytes: github.MaxResponseBytes(),
	}, key: key, clock: time.Now}, nil
}

func forgeClientTrusting(trusted *x509.CertPool) *http.Client {
	return &http.Client{
		Timeout:       domain.OutboundCallTimeout,
		CheckRedirect: refuseRedirect,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: trusted, MinVersion: tls.VersionTLS12},
		},
	}
}

func refuseRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

type installationForge struct {
	source     github.ChangeSource
	checkRuns  github.CheckRuns
	contents   github.FileContents
	dispatches github.PlanDispatches
}

func (app appForge) installationForgeFor(installation int64) (installationForge, error) {
	settings, err := app.settingsFor(installation)
	if err != nil {
		return installationForge{}, err
	}
	source, sourceErr := github.ParseChangeSource(settings)
	checkRuns, checkRunsErr := github.ParseCheckRuns(settings)
	contents, contentsErr := github.ParseFileContents(settings, domain.DeclaredLimits().MaxSourceBytes)
	dispatches, dispatchesErr := github.ParsePlanDispatches(settings)
	if err := errors.Join(sourceErr, checkRunsErr, contentsErr, dispatchesErr); err != nil {
		return installationForge{}, refusedInstallation(installation, err)
	}
	return installationForge{source: source, checkRuns: checkRuns, contents: contents, dispatches: dispatches}, nil
}

func (app appForge) settingsFor(installation int64) (github.Settings, error) {
	tokens, err := github.ParseInstallationTokens(app.settings,
		github.Installation{Key: app.key, ID: installation, Clock: app.clock})
	if err != nil {
		return github.Settings{}, refusedInstallation(installation, err)
	}
	settings := app.settings
	settings.Credentials = tokens
	return settings, nil
}

func refusedInstallation(installation int64, reason error) error {
	return fmt.Errorf("installation %d: %w", installation, reason)
}
