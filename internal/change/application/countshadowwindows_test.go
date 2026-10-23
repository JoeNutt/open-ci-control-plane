// SPDX-License-Identifier: GPL-3.0-or-later

package application_test

import (
	"context"
	"slices"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type heldRuns []domain.ShadowedRun

func (runs heldRuns) ShadowedRuns(context.Context) ([]domain.ShadowedRun, error) { return runs, nil }

type heldEscapes []domain.Escape

func (escapes heldEscapes) RecordedEscapes(context.Context) ([]domain.Escape, error) {
	return escapes, nil
}

func TestTheWindowsAreCountedFromTheRunsAndEscapesTheStoresHold(t *testing.T) {
	inShadow := domain.PlanModes{Selection: domain.TestModeShadow, Minimisation: domain.TestModeOff}
	runs := heldRuns{
		{Plan: domain.DispatchedPlan{Locator: gatedLocator(t), Modes: inShadow, Recorded: domain.RecordedSets{IsRecorded: true}}, IsReported: true},
		{Plan: domain.DispatchedPlan{Locator: gatedLocator(t), Modes: inShadow, Recorded: domain.RecordedSets{IsRecorded: true}}, IsReported: false},
	}
	escapes := heldEscapes{{Mode: domain.ShadowedSelection, PullRequest: gatedLocator(t)}}

	windows, err := application.CountShadowWindows{Runs: runs, Escapes: escapes}.Execute(context.Background(), gatedLocator(t).GovernedRepository())

	want := domain.ShadowWindowsFor(gatedLocator(t).GovernedRepository(), runs, escapes)
	if err != nil || !slices.Equal(windows, want) || windows[0].Window.Gaps != 1 || windows[0].Window.Escapes != 1 {
		t.Errorf("counted %+v (%v), want %+v with one gap and one escape against selection", windows, err, want)
	}
}
