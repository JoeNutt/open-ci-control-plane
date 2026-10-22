// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/configuration"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/random"
)

func TestAnAppKeyThatIsNotAnRSAKeyIsRefusedNamingTheAppKeyFile(t *testing.T) {
	configured := configurationWith(t, func(stated map[string]json.RawMessage) {
		stated["appKeyFile"] = quoted(t, writtenFile(t, t.TempDir(), "app-key.pem", unparseableAppKey))
	})

	_, err := appForgeOf(configured, forgeEndpoint{baseURL: forgeBaseURL, trusted: x509.NewCertPool()})

	if !errors.Is(err, configuration.ErrInvalidItem) || !strings.Contains(err.Error(), "appKeyFile") {
		t.Errorf("the refusal is not an invalid appKeyFile: %v", err)
	}
}

func configurationWith(t *testing.T, state func(map[string]json.RawMessage)) configuration.Configuration {
	t.Helper()
	stated := completeConfiguration(t)
	state(stated)
	configured, err := configuration.Read(configurationFile(t, stated))
	if err != nil {
		t.Fatalf("read the configuration: %v", err)
	}
	return configured
}

func tokenMintedFor(installation int64) string {
	return "ghs_minted-for-installation-" + strconv.FormatInt(installation, 10)
}

type recordingForge struct {
	t            *testing.T
	mutex        sync.Mutex
	exchanged    []string
	presented    []string
	listedForApp []string
	filesMovedTo string
	grantFailure int
}

func (recording *recordingForge) routes() http.Handler {
	routes := http.NewServeMux()
	routes.HandleFunc("POST /app/installations/{installation}/access_tokens", recording.grant)
	routes.HandleFunc("GET /repos/{owner}/{name}/pulls/{number}/files", recording.files)
	routes.HandleFunc("GET /repos/{owner}/{name}/commits/{sha}/check-runs", recording.checkRunListing)
	routes.HandleFunc("POST /repos/{owner}/{name}/check-runs", recording.checkRunCreation)
	return routes
}

func (recording *recordingForge) checkRunListing(answer http.ResponseWriter, asked *http.Request) {
	recording.record(&recording.presented, asked.Header.Get("Authorization"))
	recording.record(&recording.listedForApp, asked.URL.Query().Get("app_id"))
	recording.write(answer, `{"total_count":0,"check_runs":[]}`)
}

func (recording *recordingForge) checkRunCreation(answer http.ResponseWriter, asked *http.Request) {
	recording.record(&recording.presented, asked.Header.Get("Authorization"))
	answer.WriteHeader(http.StatusCreated)
	recording.write(answer, `{"id":1}`)
}

func (recording *recordingForge) grant(answer http.ResponseWriter, asked *http.Request) {
	recording.record(&recording.exchanged, asked.PathValue("installation"))
	if recording.grantFailure != 0 {
		answer.WriteHeader(recording.grantFailure)
		recording.write(answer, fmt.Sprintf(`{"message":"refused %s","token":%q}`,
			asked.Header.Get("Authorization"), echoedToken))
		return
	}
	answer.WriteHeader(http.StatusCreated)
	installation, err := strconv.ParseInt(asked.PathValue("installation"), 10, 64)
	if err != nil {
		recording.t.Errorf("a token was asked for installation %q: %v", asked.PathValue("installation"), err)
	}
	recording.write(answer, fmt.Sprintf(`{"token":%q,"expires_at":%q}`,
		tokenMintedFor(installation), time.Now().Add(time.Hour).Format(time.RFC3339)))
}

func (recording *recordingForge) files(answer http.ResponseWriter, asked *http.Request) {
	recording.record(&recording.presented, asked.Header.Get("Authorization"))
	if recording.filesMovedTo != "" {
		http.Redirect(answer, asked, recording.filesMovedTo+asked.URL.RequestURI(), http.StatusFound)
		return
	}
	recording.write(answer, `[]`)
}

func (recording *recordingForge) record(into *[]string, seen string) {
	recording.mutex.Lock()
	defer recording.mutex.Unlock()
	*into = append(*into, seen)
}

func (recording *recordingForge) write(answer http.ResponseWriter, body string) {
	if _, err := answer.Write([]byte(body)); err != nil {
		recording.t.Errorf("write the forge's answer: %v", err)
	}
}

func TestAnInstallationForgeReadsTheChangeWithATokenMintedForItsInstallation(t *testing.T) {
	recording := &recordingForge{t: t}
	installed := installationForgeAt(t, recording)
	if _, err := installed.source.FilesChanged(context.Background(), recordedPullRequest(t)); err != nil {
		t.Fatalf("read the changed files: %v", err)
	}

	if want := []string{strconv.Itoa(installationID)}; !slices.Equal(recording.exchanged, want) {
		t.Errorf("tokens were minted for installations %v, want %v", recording.exchanged, want)
	}
	if want := []string{"Bearer " + tokenMintedFor(installationID)}; !slices.Equal(recording.presented, want) {
		t.Errorf("the change was read presenting %v, want %v", recording.presented, want)
	}
}

func TestAnInstallationForgePublishesCheckRunsWithTheSameMintedToken(t *testing.T) {
	recording := &recordingForge{t: t}
	installed := installationForgeAt(t, recording)

	publishCheckRun(t, installed)

	if want := []string{strconv.Itoa(installationID)}; !slices.Equal(recording.exchanged, want) {
		t.Errorf("tokens were minted for installations %v, want %v", recording.exchanged, want)
	}
	presented := []string{"Bearer " + tokenMintedFor(installationID), "Bearer " + tokenMintedFor(installationID)}
	if !slices.Equal(recording.presented, presented) {
		t.Errorf("the check run was published presenting %v, want %v", recording.presented, presented)
	}
}

func TestTheAppForgeCallsTheConfiguredForgeAsTheConfiguredApp(t *testing.T) {
	recording := &recordingForge{t: t}

	publishCheckRun(t, recordingAppForge(t, recording).forgeFor(t, installationID))

	if want := []string{strconv.Itoa(appID)}; !slices.Equal(recording.listedForApp, want) {
		t.Errorf("check runs were listed for Apps %v, want %v", recording.listedForApp, want)
	}
}

func publishCheckRun(t *testing.T, installed installationForge) {
	t.Helper()
	head, err := domain.ParseCommitSHA("b0b0000000000000000000000000000000000b0b")
	if err != nil {
		t.Fatal(err)
	}
	id, err := random.DecisionIDs{}.NextDecisionID()
	if err != nil {
		t.Fatal(err)
	}

	err = installed.checkRuns.PublishCheckRun(context.Background(), recordedPullRequest(t), head, domain.CheckRun{
		Name: "structural", Conclusion: domain.ConclusionNeutral, Title: "No policy loaded",
		Headline: "No policy is loaded for this family.", DecisionID: id,
	})
	if err != nil {
		t.Fatalf("publish the check run: %v", err)
	}
}

func installationForgeAt(t *testing.T, recording *recordingForge) installationForge {
	t.Helper()
	return recordingAppForge(t, recording).forgeFor(t, installationID)
}

func recordingServer(t *testing.T, recording *recordingForge) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(recording.routes())
	t.Cleanup(server.Close)
	return server
}

func recordingAppForge(t *testing.T, recording *recordingForge) appForge {
	t.Helper()
	return appForgeAt(t, recordingServer(t, recording))
}

func appForgeAt(t *testing.T, server *httptest.Server) appForge {
	t.Helper()
	trusted := x509.NewCertPool()
	trusted.AddCert(server.Certificate())
	app, err := appForgeOf(configurationWith(t, func(map[string]json.RawMessage) {}), forgeEndpoint{baseURL: server.URL, trusted: trusted})
	if err != nil {
		t.Fatalf("build the App forge: %v", err)
	}
	return app
}

func (app appForge) forgeFor(t *testing.T, installation int64) installationForge {
	t.Helper()
	installed, err := app.installationForgeFor(installation)
	if err != nil {
		t.Fatalf("build the forge of installation %d: %v", installation, err)
	}
	return installed
}

func TestEachInstallationForgeMintsAndPresentsOnlyItsOwnToken(t *testing.T) {
	const otherInstallation = installationID + 1
	recording := &recordingForge{t: t}
	app := recordingAppForge(t, recording)
	own, other := app.forgeFor(t, installationID), app.forgeFor(t, otherInstallation)

	for _, installed := range []installationForge{own, other} {
		if _, err := installed.source.FilesChanged(context.Background(), recordedPullRequest(t)); err != nil {
			t.Fatalf("read the changed files: %v", err)
		}
	}

	exchanged := []string{strconv.Itoa(installationID), strconv.Itoa(otherInstallation)}
	if !slices.Equal(recording.exchanged, exchanged) {
		t.Errorf("tokens were minted for installations %v, want %v", recording.exchanged, exchanged)
	}
	presented := []string{"Bearer " + tokenMintedFor(installationID), "Bearer " + tokenMintedFor(otherInstallation)}
	if !slices.Equal(recording.presented, presented) {
		t.Errorf("the two forges presented %v, want %v", recording.presented, presented)
	}
}

func recordedPullRequest(t *testing.T) domain.PullRequestLocator {
	t.Helper()
	locator, err := domain.ParsePullRequestLocator("JoeNutt/open-ci-control-plane", 7)
	if err != nil {
		t.Fatal(err)
	}
	return locator
}

var generatedAppKey = sync.OnceValues(func() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), nil
})

func rsaAppKeyPEM(t *testing.T) string {
	t.Helper()
	encoded, err := generatedAppKey()
	if err != nil {
		t.Fatalf("generate an App key: %v", err)
	}
	return string(encoded)
}

func TestTheForgeClientDoesNotFollowARedirectToAnotherHost(t *testing.T) {
	var askedElsewhere atomic.Int32
	elsewhere := httptest.NewTLSServer(http.HandlerFunc(func(answer http.ResponseWriter, _ *http.Request) {
		askedElsewhere.Add(1)
		answer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(elsewhere.Close)
	recording := &recordingForge{t: t, filesMovedTo: elsewhere.URL}

	_, err := installationForgeAt(t, recording).source.FilesChanged(context.Background(), recordedPullRequest(t))

	if err == nil {
		t.Error("a redirected read of the changed files succeeded, want a refusal")
	}
	if asked := askedElsewhere.Load(); asked != 0 {
		t.Errorf("the other host was asked %d times, want never", asked)
	}
}

const echoedToken = "ghs_echoed-by-a-failing-forge"

func TestAFailedTokenExchangeCarriesNoTokenOrKeyInItsErrorUnderAnyVerb(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			recording := &recordingForge{t: t, grantFailure: status}

			_, err := installationForgeAt(t, recording).source.FilesChanged(context.Background(), recordedPullRequest(t))
			if err == nil {
				t.Fatal("a failed token exchange read the change, want a refusal")
			}

			keyLines := strings.Split(rsaAppKeyPEM(t), "\n")
			for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x"} {
				printed := fmt.Sprintf(verb, err)
				for name, secret := range map[string]string{
					"the echoed token": echoedToken, "the App JWT": "Bearer ey", "the App key": keyLines[1],
				} {
					if strings.Contains(printed, secret) || strings.Contains(printed, fmt.Sprintf("%x", secret)) {
						t.Errorf("under %s the error carries %s: %s", verb, name, printed)
					}
				}
			}
		})
	}
}
