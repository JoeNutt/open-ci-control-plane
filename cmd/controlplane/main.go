// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
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
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(arguments []string, logs io.Writer) int {
	log := slog.New(slog.NewJSONHandler(logs, nil))
	path, err := configurationPathIn(arguments)
	if err != nil {
		log.Error(refusedUsage, slog.String(reasonKey, err.Error()))
		return exitMisused
	}
	if _, err := configuration.Read(path); err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	return exitServed
}

func configurationPathIn(arguments []string) (string, error) {
	if len(arguments) != 1 {
		return "", errNotOnePath
	}
	return arguments[0], nil
}
