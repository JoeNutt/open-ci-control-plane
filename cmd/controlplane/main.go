// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
)

const (
	exitServed  = 0
	exitRefused = 1
)

const (
	refusedConfiguration = "configuration refused"
	reasonKey            = "reason"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(arguments []string, logs io.Writer) int {
	log := slog.New(slog.NewJSONHandler(logs, nil))
	if len(arguments) != 1 {
		log.Error("usage refused", "reason", "the control plane takes one configuration file path")
		return 2
	}
	if _, err := configuration.Read(arguments[0]); err != nil {
		log.Error(refusedConfiguration, slog.String(reasonKey, err.Error()))
		return exitRefused
	}
	return exitServed
}
