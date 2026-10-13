// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var ErrUnexpectedStatus = errors.New("the forge answered with an unexpected status")

var ErrResponseTooLarge = errors.New("the forge answered with more than the response byte bound")

func jsonAt[Payload any](
	ctx context.Context, source ChangeSource, endpoint string,
) (Payload, error) {
	var payload Payload
	request, err := requestFor(ctx, source, endpoint)
	if err != nil {
		return payload, err
	}
	response, err := source.Client.Do(request)
	if err != nil {
		return payload, fmt.Errorf("read: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if failure := failureIn(response); failure != nil {
		return payload, failure
	}
	body, err := bodyWithinBound(response.Body, source.MaxResponseBytes)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, fmt.Errorf("decode: %w", err)
	}
	return payload, nil
}

func requestFor(
	ctx context.Context, source ChangeSource, endpoint string,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, source.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	request.Header.Set("Authorization", "Bearer "+string(source.Token))
	return request, nil
}

var reportedByStatus = map[int]error{
	http.StatusNotFound: domain.ErrNoSuchPullRequest,
}

func failureIn(response *http.Response) error {
	if response.StatusCode == http.StatusOK {
		return nil
	}
	reported, mapped := reportedByStatus[response.StatusCode]
	if !mapped {
		reported = ErrUnexpectedStatus
	}
	return fmt.Errorf("%w: %s", reported, response.Status)
}

func bodyWithinBound(body io.Reader, maxBytes int) ([]byte, error) {
	read, err := io.ReadAll(io.LimitReader(body, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if len(read) > maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrResponseTooLarge, maxBytes)
	}
	return read, nil
}
