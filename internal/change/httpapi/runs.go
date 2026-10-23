// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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
	if refused := reportLengthRefusalOf(asked.ContentLength); refused != nil {
		receiving.answering.write(asked.Context(), answer, refused)
		return
	}
	asked.Body = http.MaxBytesReader(answer, asked.Body, domain.MaxRunReportBytes)
	token, err := bearerTokenIn(asked.Header.Get(authorizationHeader))
	if err != nil {
		receiving.answering.write(asked.Context(), answer, unverifiedTokenProblem())
		return
	}
	var wire struct {
		DecisionID string `json:"decisionId"`
		Tests      []struct {
			Package string `json:"package"`
			Name    string `json:"name"`
			Outcome string `json:"outcome"`
		} `json:"tests"`
	}
	decoder := json.NewDecoder(asked.Body)
	err = decoder.Decode(&wire)
	if err == nil {
		if _, trailing := decoder.Token(); trailing != io.EOF {
			err = errors.New("more than one JSON value")
		}
	}
	if err != nil {
		var overBound *http.MaxBytesError
		if errors.As(err, &overBound) {
			receiving.answering.write(asked.Context(), answer, oversizedReportProblem())
			return
		}
		receiving.answering.write(asked.Context(), answer, fieldProblem{Code: "malformed_report", Message: "the body is not one C-13 report object", Field: "body"})
		return
	}
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
