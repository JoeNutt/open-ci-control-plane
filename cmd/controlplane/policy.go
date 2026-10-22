// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/engine/opa"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

const (
	forgeBaseURL          = "https://api.github.com"
	bundleListenerAddress = "127.0.0.1:0"
	bundleServiceScheme   = "http://"
)

func servedPolicy(
	ctx context.Context, configured configuration.Configuration, evaluation evaluating,
) (opa.BundleService, <-chan struct{}, error) {
	active, log := evaluation.active, evaluation.log
	reader, err := policyReaderFor(configured, evaluation.app, active)
	if err != nil {
		return opa.BundleService{}, nil, err
	}
	service, err := servedBundles(ctx, policy.BundleHandler{Active: active}, log)
	if err != nil {
		return opa.BundleService{}, nil, err
	}
	ticker := time.NewTicker(policy.PollInterval)
	polled := make(chan struct{})
	go func() {
		defer close(polled)
		defer ticker.Stop()
		reader.PollAtEachTick(ctx, ticker.C)
	}()
	log.Info(policyRead, slog.String(policyRepositoryKey, configured.PolicyRepository().String()))
	return service, polled, nil
}

func policyReaderFor(
	configured configuration.Configuration, app appForge, active *policy.ActivePolicy,
) (*policy.Reader, error) {
	settings, err := app.settingsFor(configured.PolicyInstallationID())
	if err != nil {
		return nil, err
	}
	ports, err := policyPortsOver(settings)
	if err != nil {
		return nil, err
	}
	return policy.ParseReader(configured.PolicyRepository(), ports, active)
}

func policyPortsOver(settings github.Settings) (policy.ReaderPorts, error) {
	heads, err := github.ParseDefaultBranchHeads(settings)
	if err != nil {
		return policy.ReaderPorts{}, err
	}
	listings, err := github.ParsePolicyListings(settings)
	if err != nil {
		return policy.ReaderPorts{}, err
	}
	contents, err := github.ParsePolicyContents(settings)
	if err != nil {
		return policy.ReaderPorts{}, err
	}
	return policy.ReaderPorts{Heads: heads, Listings: listings, Contents: contents}, nil
}

func servedBundles(ctx context.Context, bundles http.Handler, log *slog.Logger) (opa.BundleService, error) {
	listener, err := net.Listen("tcp", bundleListenerAddress)
	if err != nil {
		return opa.BundleService{}, err
	}
	server := &http.Server{Handler: bundles, ReadHeaderTimeout: readHeaderBound}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(failedBundleListener, slog.String(reasonKey, err.Error()))
		}
	}()
	go func() {
		<-ctx.Done()
		if err := server.Close(); err != nil {
			log.Error(failedBundleListener, slog.String(reasonKey, err.Error()))
		}
	}()
	return opa.ParseBundleService(bundleServiceScheme + listener.Addr().String())
}
