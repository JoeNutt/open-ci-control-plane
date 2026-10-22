// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

const forgeToken = "s3cret"

var forgeCredential = github.NewCredential(forgeToken)

const forgeAppID = 4242

const forgeTimeout = 5 * time.Second

func locatorOf(t *testing.T, repository string, number int) domain.PullRequestLocator {
	t.Helper()

	locator, err := domain.ParsePullRequestLocator(repository, number)
	if err != nil {
		t.Fatalf("parse pull request locator: %v", err)
	}
	return locator
}

func forge(t *testing.T, answer http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(answer)
	t.Cleanup(server.Close)
	return server
}

func serving(t *testing.T, path, body string) *httptest.Server {
	t.Helper()

	return servingEach(t, map[string]string{path: body})
}

func servingEach(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, request *http.Request) {
		body, served := bodies[request.URL.Path]
		if !served {
			t.Errorf("the adapter asked for %s, which this forge does not serve",
				request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		write(t, writer, body)
	})
}

func answering(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	return forge(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(status)
		write(t, writer, body)
	})
}

func write(t *testing.T, writer http.ResponseWriter, body string) {
	t.Helper()

	if _, written := writer.Write([]byte(body)); written != nil {
		t.Errorf("write response: %v", written)
	}
}

func wellFormedSettings() github.Settings {
	return github.Settings{
		BaseURL:          originAt("https://api.example"),
		Credentials:      forgeCredential,
		AppID:            forgeAppID,
		Client:           &http.Client{Timeout: forgeTimeout},
		MaxResponseBytes: 1 << 20,
	}
}

func sourceAt(t *testing.T, server *httptest.Server) github.ChangeSource {
	t.Helper()

	return sourceWith(t, settingsAt(server))
}

func settingsAt(server *httptest.Server) github.Settings {
	settings := wellFormedSettings()
	settings.BaseURL = originAt(server.URL)
	settings.Client = server.Client()
	settings.Client.Timeout = forgeTimeout
	return settings
}

func sourceWith(t *testing.T, settings github.Settings) github.ChangeSource {
	t.Helper()

	source, err := github.ParseChangeSource(settings)
	if err != nil {
		t.Fatalf("parse change source: %v", err)
	}
	return source
}

func pageAsLongAsAsked(t *testing.T, request *http.Request) string {
	t.Helper()

	asked := request.URL.Query().Get("per_page")
	length, err := strconv.Atoi(asked)
	if err != nil {
		t.Errorf("the adapter asked for per_page=%q: %v", asked, err)
		return "[]"
	}
	listings := make([]string, 0, length)
	for position := range length {
		listings = append(listings,
			fmt.Sprintf(`{"filename": "internal/%d.go", "status": "added"}`, position))
	}
	return "[" + strings.Join(listings, ",") + "]"
}

func originAt(address string) domain.HTTPSOrigin {
	origin, err := domain.ParseHTTPSOrigin(address)
	if err != nil {
		panic(fmt.Sprintf("the fixture address %q is not an https origin: %v", address, err))
	}
	return origin
}
