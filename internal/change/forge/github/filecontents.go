// SPDX-License-Identifier: GPL-3.0-or-later

package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

var _ domain.FileContents = FileContents{}

var errNotAFile = errors.New("the forge answered for something that is not a file")

var errContentsNotServed = errors.New("the forge answered without the file's content")

var ErrMalformedFileBound = errors.New("the file byte bound is not positive")

var errContentsSizeMismatch = errors.New("the forge served content of another size than it states")

type FileContents struct {
	forge        connection
	maxFileBytes int
}

func ParseFileContents(settings Settings, maxFileBytes int) (FileContents, error) {
	if maxFileBytes <= 0 {
		return FileContents{}, fmt.Errorf("%w: %d", ErrMalformedFileBound, maxFileBytes)
	}
	forge, err := parseCredentialsConnection(settings)
	if err != nil {
		return FileContents{}, err
	}
	return FileContents{forge: forge, maxFileBytes: maxFileBytes}, nil
}

func (contents FileContents) ContentsAt(
	ctx context.Context, file domain.FileAtCommit,
) ([]byte, error) {
	read, err := contents.decodedAt(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	return read, nil
}

func (contents FileContents) decodedAt(ctx context.Context, file domain.FileAtCommit) ([]byte, error) {
	answer, err := exchanged[contentsAnswer](ctx, contents.forge, contentsReading(file))
	if err != nil {
		return nil, refusalOfAListing(err)
	}
	return answer.decoded(contents.maxFileBytes)
}

const jsonArrayValue = "array"

func refusalOfAListing(err error) error {
	var mistyped *json.UnmarshalTypeError
	if errors.As(err, &mistyped) && mistyped.Value == jsonArrayValue && mistyped.Field == "" {
		return fmt.Errorf("%w: a directory listing", errNotAFile)
	}
	return err
}

func contentsReading(file domain.FileAtCommit) exchange {
	return exchange{
		method:    http.MethodGet,
		endpoint:  contentsEndpoint(file),
		succeeded: http.StatusOK,
		notFound:  domain.ErrNoSuchFile,
	}
}

func contentsEndpoint(file domain.FileAtCommit) string {
	query := url.Values{"ref": {file.Commit().String()}}
	return fmt.Sprintf("%s/contents/%s?%s",
		repositoryEndpoint(file.Repository()), escapedPathOf(file.Path()), query.Encode())
}

func escapedPathOf(path domain.RepositoryPath) string {
	segments := path.Segments()
	for position, segment := range segments {
		segments[position] = url.PathEscape(segment)
	}
	return strings.Join(segments, urlPathSeparator)
}

type contentsAnswer struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

const (
	contentsTypeFile       = "file"
	contentsEncodingBase64 = "base64"
)

func (answer contentsAnswer) decoded(maxFileBytes int) ([]byte, error) {
	if refusal := answer.refusalBeforeDecoding(maxFileBytes); refusal != nil {
		return nil, refusal
	}
	decoded, err := base64.StdEncoding.DecodeString(answer.Content)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(decoded) != answer.Size {
		return nil, fmt.Errorf("%w: %d bytes served, %d stated", errContentsSizeMismatch, len(decoded), answer.Size)
	}
	return decoded, nil
}

func (answer contentsAnswer) refusalBeforeDecoding(maxFileBytes int) error {
	if answer.Type != contentsTypeFile {
		return fmt.Errorf("%w: a %q", errNotAFile, answer.Type)
	}
	if answer.Size > maxFileBytes {
		return fmt.Errorf("%w: %d bytes exceeds %d", domain.ErrSourceTooLarge, answer.Size, maxFileBytes)
	}
	if answer.Encoding != contentsEncodingBase64 {
		return fmt.Errorf("%w: encoding %q", errContentsNotServed, answer.Encoding)
	}
	return nil
}
