// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"bytes"
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

type exchange struct {
	method    string
	endpoint  string
	sent      any
	succeeded int
	notFound  error
}

func jsonAt[Payload any](
	ctx context.Context, forge connection, endpoint string,
) (Payload, error) {
	return exchanged[Payload](ctx, forge, exchange{
		method:    http.MethodGet,
		endpoint:  endpoint,
		succeeded: http.StatusOK,
	})
}

func exchanged[Payload any](
	ctx context.Context, forge connection, over exchange,
) (Payload, error) {
	var payload Payload
	response, err := answered(ctx, forge, over)
	if err != nil {
		return payload, err
	}
	defer func() { _ = response.Body.Close() }()

	body, err := bodyWithinBound(response.Body, forge.settings.MaxResponseBytes)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, fmt.Errorf("decode: %w", err)
	}
	return payload, nil
}

func delivered(ctx context.Context, forge connection, over exchange) error {
	response, err := answered(ctx, forge, over)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	// The write has already succeeded; the body is drained, bounded, only so the client
	// can reuse the connection: https://pkg.go.dev/net/http#Client.Do
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, int64(forge.settings.MaxResponseBytes)))
	return nil
}

func answered(ctx context.Context, forge connection, over exchange) (*http.Response, error) {
	request, err := requestFor(ctx, forge, over)
	if err != nil {
		return nil, err
	}
	response, err := forge.settings.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send %s %s: %w", over.method, over.endpoint, err)
	}
	if failure := failureIn(response, over); failure != nil {
		_ = response.Body.Close()
		return nil, failure
	}
	return response, nil
}

func requestFor(
	ctx context.Context, forge connection, over exchange,
) (*http.Request, error) {
	body, err := encodedBodyOf(over)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(
		ctx, over.method, forge.settings.BaseURL+over.endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", over.endpoint, err)
	}
	request.Header.Set("Authorization", forge.token.bearer())
	return request, nil
}

func encodedBodyOf(over exchange) (io.Reader, error) {
	if over.sent == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(over.sent)
	if err != nil {
		return nil, fmt.Errorf("encode %s %s: %w", over.method, over.endpoint, err)
	}
	return bytes.NewReader(encoded), nil
}

var reportedByStatus = map[int]error{
	http.StatusNotFound: domain.ErrNoSuchPullRequest,
}

func failureIn(response *http.Response, over exchange) error {
	if response.StatusCode == over.succeeded {
		return nil
	}
	if response.StatusCode == http.StatusNotFound && over.notFound != nil {
		return fmt.Errorf("%w: %s", over.notFound, response.Status)
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
