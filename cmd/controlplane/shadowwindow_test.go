// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type heldShadowRuns []domain.ShadowedRun

func (runs heldShadowRuns) ShadowedRuns(context.Context) ([]domain.ShadowedRun, error) {
	return runs, nil
}

type heldEscapeHistory []domain.Escape

func (escapes heldEscapeHistory) RecordedEscapes(context.Context) ([]domain.Escape, error) {
	return escapes, nil
}

func TestEachModesShadowWindowIsLoggedWithItsRunsGapsAndEscapes(t *testing.T) {
	locator := recordedPullRequest(t)
	inShadow := domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeShadow}
	counting := application.CountShadowWindows{
		Runs:    heldShadowRuns{{Plan: domain.DispatchedPlan{Locator: locator, Modes: inShadow, Recorded: domain.RecordedSets{IsRecorded: true}}}},
		Escapes: heldEscapeHistory{{Mode: domain.ShadowedSelection, PullRequest: locator}},
	}
	var logged bytes.Buffer

	logShadowWindows(context.Background(), slog.New(slog.NewJSONHandler(&logged, nil)), counting, locator.GovernedRepository())

	lines := strings.Split(strings.TrimSpace(logged.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want one per mode: %s", len(lines), logged.String())
	}
	var selection map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &selection); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if selection["msg"] != "shadow window counted" || selection["repository"] != locator.Repository() || selection["mode"] != "selection" ||
		selection["runs"] != 1.0 || selection["gaps"] != 1.0 || selection["escapes"] != 1.0 {
		t.Errorf("logged %v, want selection's window for %s: 1 run, 1 gap, 1 escape", selection, locator.Repository())
	}
}
