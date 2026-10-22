// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/jsonlines"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/random"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/rules"
)

const (
	exitServed  = 0
	exitRefused = 1
	exitMisused = 2
)

const (
	readHeaderBound  = 10 * time.Second
	shutdownBound    = 10 * time.Second
	readBound        = 30 * time.Second
	engineStartBound = 30 * time.Second
)

var errNotOnePath = errors.New("the control plane takes one configuration file path")

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stderr))
}

func run(ctx context.Context, arguments []string, logs io.Writer) int {
	return runAgainst(ctx, arguments, logs, githubForge)
}

func githubForge() (forgeEndpoint, error) {
	trusted, err := x509.SystemCertPool()
	return forgeEndpoint{baseURL: forgeBaseURL, trusted: trusted}, err
}

func runAgainst(ctx context.Context, arguments []string, logs io.Writer, endpointOf func() (forgeEndpoint, error)) int {
	log := slog.New(slog.NewJSONHandler(logs, nil))
	path, err := configurationPathIn(arguments)
	if err != nil {
		log.Error(refusedUsage, slog.String(reasonKey, err.Error()))
		return exitMisused
	}
	configured, err := configuration.Read(path)
	if err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	endpoint, err := endpointOf()
	if err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	app, err := appForgeOf(configured, endpoint)
	if err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	log.Info(readConfiguration, slog.Any(familiesKey, configured.Families()),
		slog.Any(installationsKey, configured.InstallationIDs()))
	serving, stopServing := context.WithCancel(ctx)
	engines, polled, err := startedPolicyEngines(serving, configured, app, log)
	defer func() {
		stopServing()
		<-polled
	}()
	if err != nil {
		log.Error(refusedEngines, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	defer engines.stop(context.WithoutCancel(ctx))
	gate, err := engines.gate()
	if err != nil {
		log.Error(refusedEngines, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	rules, err := configuration.ClassificationRulesParsedBy(configured, rulesIn)
	if err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	recorder, err := jsonlines.ParseDecisionRecorder(os.Stdout)
	if err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	server, err := serverFor(configured, gatesFor(configured.InstallationIDs(),
		evaluating{app: app, gate: gate, rules: rules, recorder: recorder, log: log}), log)
	if err != nil {
		log.Error(refusedListener, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	return servedUntil(ctx, server, log)
}

func startedPolicyEngines(
	ctx context.Context, configured configuration.Configuration, app appForge, log *slog.Logger,
) (startedEngines, <-chan struct{}, error) {
	service, polled, err := servedPolicy(ctx, configured, app, log)
	if err != nil {
		return nil, closedSignal(), err
	}
	starting, cancel := context.WithTimeout(ctx, engineStartBound)
	defer cancel()
	engines, err := startedEnginesFor(starting, configured.Families(), bundledEngineStarter(service))
	if err != nil {
		return nil, polled, err
	}
	log.Info(enginesStarted, slog.Any(familiesKey, configured.Families()))
	return engines, polled, nil
}

func closedSignal() <-chan struct{} {
	closed := make(chan struct{})
	close(closed)
	return closed
}

func configurationPathIn(arguments []string) (string, error) {
	if len(arguments) != 1 {
		return "", errNotOnePath
	}
	return arguments[0], nil
}

func rulesIn(content []byte) (domain.ClassificationRules, error) {
	return rules.ReadFrom(bytes.NewReader(content))
}

func serverFor(
	configured configuration.Configuration, gates map[int64]httpapi.ChangeGate, log *slog.Logger,
) (*http.Server, error) {
	routes, err := httpapi.ParseRoutes(httpapi.Settings{
		Log:         log,
		Secret:      domain.WebhookSecret(configured.WebhookSecret().Reveal()),
		Gates:       gates,
		DecisionIDs: random.DecisionIDs{},
		Clock:       time.Now,
	})
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:              configured.ListenAddress().String(),
		Handler:           routes,
		ReadHeaderTimeout: readHeaderBound,
		ReadTimeout:       readBound,
	}, nil
}

func servedUntil(ctx context.Context, server *http.Server, log *slog.Logger) int {
	stopped := make(chan error, 1)
	go func() {
		<-ctx.Done()
		stopping, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownBound)
		defer cancel()
		stopped <- server.Shutdown(stopping)
	}()
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error(failedListener, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	if err := <-stopped; err != nil {
		log.Error(failedShutdown, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	return exitServed
}
