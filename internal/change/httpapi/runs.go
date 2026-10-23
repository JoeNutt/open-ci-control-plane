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
	target, refused := runTargetOf(asked.PathValue(ownerSegment), asked.PathValue(repoSegment), asked.PathValue(headSegment))
	if refused != nil {
		receiving.answering.write(asked.Context(), answer, refused)
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
	report, _ := domain.ParseRunReport(target.repository, target.head, id, results)
	_ = receiving.recording.Execute(asked.Context(), application.RunUpload{Token: token, Report: report})
	receiving.answering.write(asked.Context(), answer, runAccepted{HeadSHA: target.head.String(), Recorded: len(results)})
}

const headSegment = "headSha"

type runTarget struct {
	repository domain.GovernedRepository
	head       domain.CommitSHA
}

func runTargetOf(owner, repo, head string) (runTarget, refusal) {
	repository, err := domain.ParseGovernedRepositorySegments(owner, repo)
	if err != nil {
		return runTarget{}, malformedPathProblem(repositorySegments)
	}
	parsedHead, err := domain.ParseCommitSHA(head)
	if err != nil {
		return runTarget{}, malformedPathProblem(headSegment)
	}
	return runTarget{repository: repository, head: parsedHead}, nil
}

type runAccepted struct {
	HeadSHA  string `json:"headSha"`
	Recorded int    `json:"recorded"`
}

func (runAccepted) status() int { return http.StatusCreated }
