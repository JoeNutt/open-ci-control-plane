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
	Log           *slog.Logger
	Secret        domain.WebhookSecret
	Installations []int64
	Gate          ChangeGate
	DecisionIDs   domain.DecisionIDSource
	Clock         func() time.Time
}

type requiredPart struct {
	name      string
	isMissing func(Settings) bool
}

var requiredParts = []requiredPart{
	{name: "a log", isMissing: func(settings Settings) bool { return settings.Log == nil }},
	{name: "a gate", isMissing: func(settings Settings) bool { return settings.Gate == nil }},
	{name: "a decision ID source", isMissing: func(settings Settings) bool { return settings.DecisionIDs == nil }},
	{name: "an allowed installation", isMissing: func(settings Settings) bool { return len(settings.Installations) == 0 }},
	{name: "a clock", isMissing: func(settings Settings) bool { return settings.Clock == nil }},
}

func isMissingFrom(settings Settings) func(requiredPart) bool {
	return func(part requiredPart) bool { return part.isMissing(settings) }
}
