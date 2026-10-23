// SPDX-License-Identifier: GPL-3.0-or-later

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	receiving.answering.write(asked.Context(), answer, receiving.answerTo(answer, asked))
}

func (receiving runReports) answerTo(answer http.ResponseWriter, asked *http.Request) answerBody {
	target, refused := runTargetOf(asked.PathValue(ownerSegment), asked.PathValue(repoSegment), asked.PathValue(headSegment))
	if refused != nil {
		return refused
	}
	if refused := reportLengthRefusalOf(asked.ContentLength); refused != nil {
		return refused
	}
	asked.Body = http.MaxBytesReader(answer, asked.Body, domain.MaxRunReportBytes)
	token, err := bearerTokenIn(asked.Header.Get(authorizationHeader))
	if err != nil {
		return unverifiedTokenProblem()
	}
	body, refused := reportBodyOf(asked.Body)
	if refused != nil {
		return refused
	}
	report, refused := body.reportFor(target)
	if refused != nil {
		return refused
	}
	_ = receiving.recording.Execute(asked.Context(), application.RunUpload{Token: token, Report: report})
	return runAccepted{HeadSHA: target.head.String(), Recorded: len(report.Results())}
}

type reportBody struct {
	DecisionID string       `json:"decisionId"`
	Tests      []reportTest `json:"tests"`
}

type reportTest struct {
	Package string `json:"package"`
	Name    string `json:"name"`
	Outcome string `json:"outcome"`
}

func (body reportBody) reportFor(target runTarget) (domain.RunReport, refusal) {
	id, err := domain.ParseDecisionID(body.DecisionID)
	if err != nil {
		return domain.RunReport{}, unparseableReportFieldProblem(decisionIDField)
	}
	results := make([]domain.TestResult, 0, len(body.Tests))
	for _, test := range body.Tests {
		result, err := domain.ParseTestResult(test.Package, test.Name, test.Outcome)
		if err != nil {
			return domain.RunReport{}, unparseableReportFieldProblem(testsField)
		}
		results = append(results, result)
	}
	report, err := domain.ParseRunReport(target.repository, target.head, id, results)
	if err != nil {
		return domain.RunReport{}, limitProblem{Code: "too_many_tests", Message: "x", Limit: fmt.Sprintf("%d tests", domain.MaxReportedTests)}
	}
	return report, nil
}

var errTrailingValue = errors.New("the body holds more than one JSON value")

func reportBodyOf(stated io.Reader) (reportBody, refusal) {
	decoder := json.NewDecoder(stated)
	var body reportBody
	err := decoder.Decode(&body)
	if err == nil {
		err = endOf(decoder)
	}
	var overBound *http.MaxBytesError
	if errors.As(err, &overBound) {
		return reportBody{}, oversizedReportProblem()
	}
	if err != nil {
		return reportBody{}, malformedReportProblem()
	}
	return body, nil
}

func endOf(decoder *json.Decoder) error {
	_, err := decoder.Token()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	return errTrailingValue
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
