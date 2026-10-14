// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.CheckRunPublisher = CheckRuns{}

type CheckRuns struct {
	forge connection
}

func ParseCheckRuns(settings Settings) (CheckRuns, error) {
	forge, err := parseConnection(settings)
	if err != nil {
		return CheckRuns{}, err
	}
	return CheckRuns{forge: forge}, nil
}

func (runs CheckRuns) PublishCheckRun(
	ctx context.Context, locator domain.PullRequestLocator, headSHA string, run domain.CheckRun,
) error {
	_, err := jsonAt[struct {
		TotalCount int `json:"total_count"`
	}](ctx, runs.forge, fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs?check_name=%s",
		locator.Owner(), locator.Name(), headSHA, url.QueryEscape(run.Name)))
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"name":       run.Name,
		"head_sha":   headSHA,
		"status":     "completed",
		"conclusion": string(run.Conclusion),
		"output": map[string]string{
			"title":   run.Title,
			"summary": run.Summary,
			"text":    run.Text,
		},
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		runs.forge.settings.BaseURL+fmt.Sprintf("/repos/%s/%s/check-runs",
			locator.Owner(), locator.Name()), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+string(runs.forge.settings.Token))
	response, err := runs.forge.settings.Client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("%w: %s", ErrUnexpectedStatus, response.Status)
	}
	return nil
}
