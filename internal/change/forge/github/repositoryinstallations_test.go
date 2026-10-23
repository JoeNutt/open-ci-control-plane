// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const installedRepository = "JoeNutt/open-ci-control-plane"

var lookedUpAt = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func TestARepositorysInstallationIsLookedUpAsTheApp(t *testing.T) {
	routes := http.NewServeMux()
	routes.HandleFunc("GET /repos/"+installedRepository+"/installation", func(answer http.ResponseWriter, asked *http.Request) {
		if !isPresentedAppJWT(t, asked, lookedUpAt) {
			answer.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = answer.Write([]byte(`{"id": ` + strconv.Itoa(forgeInstallationID) + `, "app_id": 1}`))
	})
	server := httptest.NewTLSServer(routes)
	t.Cleanup(server.Close)
	repository, err := domain.ParseGovernedRepository(installedRepository)
	if err != nil {
		t.Fatalf("parse repository: %v", err)
	}
	installations, err := github.ParseRepositoryInstallations(settingsAt(server), github.Installation{
		Key: forgeAppKey(t), Clock: clockAt(lookedUpAt).Now,
	})
	if err != nil {
		t.Fatalf("parse repository installations: %v", err)
	}

	installation, err := installations.InstallationOf(t.Context(), repository)

	if err != nil || installation != forgeInstallationID {
		t.Errorf("looked up installation %d (%v), want %d", installation, err, forgeInstallationID)
	}
}
