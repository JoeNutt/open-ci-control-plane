// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
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
	answer, err := jsonAt[contentsAnswer](ctx, contents.forge, contentsEndpoint(file))
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(answer.Content)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return decoded, nil
}

func contentsEndpoint(file domain.FileAtCommit) string {
	query := url.Values{"ref": {file.Commit.String()}}
	return fmt.Sprintf("/repos/%s/contents/%s?%s",
		file.Repository, escapedRepositoryPath(file.Path), query.Encode())
}

const repositoryPathSeparator = "/"

func escapedRepositoryPath(path string) string {
	segments := strings.Split(path, repositoryPathSeparator)
	for position, segment := range segments {
		segments[position] = url.PathEscape(segment)
	}
	return strings.Join(segments, repositoryPathSeparator)
}

type contentsAnswer struct {
	Content string `json:"content"`
}
