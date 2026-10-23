// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrIncompleteRoutes = errors.New("the listener is missing a part it cannot work without")

type ChangeGate interface {
	Execute(ctx context.Context, locator domain.PullRequestLocator, id domain.DecisionID) error
}

type Settings struct {
	Log         *slog.Logger
	Secret      domain.WebhookSecret
	Gates       map[int64]ChangeGate
	DecisionIDs domain.DecisionIDSource
	Clock       func() time.Time
	Coverage    CoverageReplacement
	Artifacts   CoverageArtifacts
	Runs        RunRecording
}

type requiredPart struct {
	name      string
	isMissing func(Settings) bool
}

var requiredParts = []requiredPart{
	{name: "a log", isMissing: func(settings Settings) bool { return settings.Log == nil }},
	{name: "a decision ID source", isMissing: func(settings Settings) bool { return settings.DecisionIDs == nil }},
	{name: "a clock", isMissing: func(settings Settings) bool { return settings.Clock == nil }},
	{name: "a coverage replacement", isMissing: func(settings Settings) bool { return settings.Coverage == nil }},
	{name: "an artifact parser", isMissing: func(settings Settings) bool { return settings.Artifacts == nil }},
}

func isMissingFrom(settings Settings) func(requiredPart) bool {
	return func(part requiredPart) bool { return part.isMissing(settings) }
}
