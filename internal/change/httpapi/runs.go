// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/application"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

const runsRoute = "PUT /v1/repositories/{owner}/{repo}/runs/{headSha}"

type RunRecording interface {
	Execute(ctx context.Context, upload application.RunUpload) error
}

type runReports struct {
	answering answers
	recording RunRecording
}

func (receiving runReports) ServeHTTP(answer http.ResponseWriter, asked *http.Request) {
	repository, err := domain.ParseGovernedRepositorySegments(asked.PathValue(ownerSegment), asked.PathValue(repoSegment))
	if err != nil {
		receiving.answering.write(asked.Context(), answer, malformedPathProblem(repositorySegments))
		return
	}
	head, err := domain.ParseCommitSHA(asked.PathValue("headSha"))
	if err != nil {
		receiving.answering.write(asked.Context(), answer, malformedPathProblem("headSha"))
		return
	}
	token, _ := bearerTokenIn(asked.Header.Get(authorizationHeader))
	var wire struct {
		DecisionID string `json:"decisionId"`
		Tests      []struct {
			Package string `json:"package"`
			Name    string `json:"name"`
			Outcome string `json:"outcome"`
		} `json:"tests"`
	}
	_ = json.NewDecoder(asked.Body).Decode(&wire)
	id, _ := domain.ParseDecisionID(wire.DecisionID)
	var results []domain.TestResult
	for _, test := range wire.Tests {
		result, _ := domain.ParseTestResult(test.Package, test.Name, test.Outcome)
		results = append(results, result)
	}
	report, _ := domain.ParseRunReport(repository, head, id, results)
	_ = receiving.recording.Execute(asked.Context(), application.RunUpload{Token: token, Report: report})
	receiving.answering.write(asked.Context(), answer, runAccepted{HeadSHA: head.String(), Recorded: len(results)})
}

type runAccepted struct {
	HeadSHA  string `json:"headSha"`
	Recorded int    `json:"recorded"`
}

func (runAccepted) status() int { return http.StatusCreated }
