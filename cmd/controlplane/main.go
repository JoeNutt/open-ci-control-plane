// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/httpapi"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/random"
)

const (
	exitServed  = 0
	exitRefused = 1
	exitMisused = 2
)

const (
	readHeaderBound = 10 * time.Second
	shutdownBound   = 10 * time.Second
)

var errNotOnePath = errors.New("the control plane takes one configuration file path")

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stderr))
}

func run(ctx context.Context, arguments []string, logs io.Writer) int {
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
	log.Info(readConfiguration, slog.Any(familiesKey, configured.Families()),
		slog.Any(installationsKey, configured.InstallationIDs()))
	server, err := serverFor(configured, log)
	if err != nil {
		log.Error(refusedListener, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	return servedUntil(ctx, server, log)
}

func configurationPathIn(arguments []string) (string, error) {
	if len(arguments) != 1 {
		return "", errNotOnePath
	}
	return arguments[0], nil
}

func serverFor(configured configuration.Configuration, log *slog.Logger) (*http.Server, error) {
	routes, err := httpapi.ParseRoutes(httpapi.Settings{
		Log:           log,
		Secret:        domain.WebhookSecret(configured.WebhookSecret().Reveal()),
		Installations: configured.InstallationIDs(),
		Gate:          unwiredEvaluation{},
		DecisionIDs:   random.DecisionIDs{},
	})
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:              configured.ListenAddress().String(),
		Handler:           routes,
		ReadHeaderTimeout: readHeaderBound,
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
