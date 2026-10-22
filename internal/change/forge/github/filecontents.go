// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type FileContents struct {
	forge    connection
	maxBytes int
}

func ParseFileContents(settings Settings, maxBytes int) (FileContents, error) {
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return FileContents{}, err
	}
	return FileContents{forge: forge, maxBytes: maxBytes}, nil
}

type contentsAnswer struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

func (contents FileContents) ContentsAt(
	ctx context.Context, file domain.FileAtCommit,
) ([]byte, error) {
	endpoint := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", file.Repository, file.Path, file.Commit)
	answer, err := jsonAt[contentsAnswer](ctx, contents.forge, endpoint)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(answer.Content)
}
