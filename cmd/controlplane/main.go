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
	refusedConfiguration = "configuration refused"
	refusedUsage         = "usage refused"
	reasonKey            = "reason"
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
	log.Info("configuration read", "families", configured.Families(), "installations", configured.InstallationIDs())
	routes, err := httpapi.ParseRoutes(httpapi.Settings{
		Log:           log,
		Secret:        domain.WebhookSecret(configured.WebhookSecret().Reveal()),
		Installations: configured.InstallationIDs(),
		Gate:          unwiredEvaluation{},
		DecisionIDs:   random.DecisionIDs{},
	})
	if err != nil {
		log.Error("listener refused", "reason", err.Error())
		return exitRefused
	}
	server := &http.Server{Addr: configured.ListenAddress().String(), Handler: routes, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("listener failed", "reason", err.Error())
		return exitRefused
	}
	return exitServed
}

var errEvaluationUnwired = errors.New("evaluation is not wired: the forge credential arrives with #120 and the policy bundle with step 4")

type unwiredEvaluation struct{}

func (unwiredEvaluation) Execute(context.Context, domain.PullRequestLocator, domain.DecisionID) error {
	return errEvaluationUnwired
}

func configurationPathIn(arguments []string) (string, error) {
	if len(arguments) != 1 {
		return "", errNotOnePath
	}
	return arguments[0], nil
}
