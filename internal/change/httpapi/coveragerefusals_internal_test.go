// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

func TestAnUploadFromAnotherRepositoryIsAnswered403(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("authorise: %w", application.ErrForeignRepository))

	if refused.answer.status() != http.StatusForbidden {
		t.Errorf("answered %#v, want %d", refused.answer, http.StatusForbidden)
	}
}

func TestAnUnverifiedTokenIsAnswered401AsARefusal(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("verify: %w", domain.ErrUnverifiedIdentityToken))

	if refused.answer.status() != http.StatusUnauthorized || refused.level != slog.LevelWarn {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusUnauthorized, slog.LevelWarn)
	}
}

func TestATokenWhoseIssuerCannotBeReadIsAnswered401AndLoggedAsOurOutage(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("verify: %w", domain.ErrIdentityIssuerUnavailable))

	if refused.answer.status() != http.StatusUnauthorized || refused.level != slog.LevelError {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusUnauthorized, slog.LevelError)
	}
}

func TestAFailureC12DoesNotClassifyIsAnswered500AndLoggedAsAnError(t *testing.T) {
	refused := coverageRefusalOf(errors.New("the store is full"))

	if refused.answer.status() != http.StatusInternalServerError || refused.level != slog.LevelError {
		t.Errorf("answered %#v at %s, want %d at %s", refused.answer, refused.level, http.StatusInternalServerError, slog.LevelError)
	}
}

func TestAnArtifactNamingAPathOutsideTheRepositoryIsAnswered422(t *testing.T) {
	refused := coverageRefusalOf(fmt.Errorf("refuse coverage: line 1: %w", domain.ErrPathOutsideRepository))

	if refused.answer.status() != http.StatusUnprocessableEntity {
		t.Errorf("answered %#v, want %d", refused.answer, http.StatusUnprocessableEntity)
	}
}
