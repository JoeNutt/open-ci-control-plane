// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, arguments []string, decisions io.Writer, logs io.Writer) int {
	log := slog.New(slog.NewJSONHandler(logs, nil))
	_, err := configuration.Read(arguments[0])
	if err != nil {
		log.Error("configuration refused", "reason", err.Error())
		return 1
	}
	return 0
}
