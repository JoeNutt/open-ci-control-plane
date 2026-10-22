// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.FileContents = FileContents{}

type FileContents struct {
	forge connection
}

func ParseFileContents(settings Settings) (FileContents, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return FileContents{}, err
	}
	return FileContents{forge: forge}, nil
}

func (contents FileContents) ContentsAt(
	ctx context.Context, file domain.FileAtCommit,
) ([]byte, error) {
	read, err := contents.decodedAt(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("read %s in %s at %s: %w", file.Path, file.Repository, file.Commit, err)
	}
	return read, nil
}

func (contents FileContents) decodedAt(ctx context.Context, file domain.FileAtCommit) ([]byte, error) {
	reading, err := contentsReading(file)
	if err != nil {
		return nil, err
	}
	answer, err := exchanged[contentsAnswer](ctx, contents.forge, reading)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(answer.Content)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return decoded, nil
}

func contentsReading(file domain.FileAtCommit) (exchange, error) {
	endpoint, err := contentsEndpoint(file)
	if err != nil {
		return exchange{}, err
	}
	return exchange{
		method:    http.MethodGet,
		endpoint:  endpoint,
		succeeded: http.StatusOK,
		notFound:  domain.ErrNoSuchFile,
	}, nil
}

func contentsEndpoint(file domain.FileAtCommit) (string, error) {
	repository, err := domain.ParseGovernedRepository(file.Repository)
	if err != nil {
		return "", err
	}
	path, err := escapedRepositoryPath(file.Path)
	if err != nil {
		return "", err
	}
	query := url.Values{"ref": {file.Commit.String()}}
	return fmt.Sprintf("%s/contents/%s?%s", repositoryEndpoint(repository), path, query.Encode()), nil
}

const repositoryPathSeparator = "/"

var segmentsResolvedAway = []string{"", ".", ".."}

func escapedRepositoryPath(path string) (string, error) {
	segments := strings.Split(path, repositoryPathSeparator)
	for position, segment := range segments {
		if slices.Contains(segmentsResolvedAway, segment) {
			return "", fmt.Errorf("%w: it holds the segment %q", domain.ErrPathOutsideRepository, segment)
		}
		segments[position] = url.PathEscape(segment)
	}
	return strings.Join(segments, repositoryPathSeparator), nil
}

type contentsAnswer struct {
	Content string `json:"content"`
}
