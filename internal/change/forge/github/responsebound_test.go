// SPDX-License-Identifier: GPL-3.0-or-later

package github_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/forge/github"
)

func TestTheResponseBoundAdmitsTheLargestSourceFileAsTheContentsAPIEncodesIt(t *testing.T) {
	limits := domain.DeclaredLimits()
	encoded := base64.StdEncoding.EncodeToString(make([]byte, limits.MaxSourceBytes))
	var lines []string
	for start := 0; start < len(encoded); start += 60 {
		lines = append(lines, encoded[start:min(start+60, len(encoded))])
	}
	path := strings.Repeat("%41", limits.MaxPathBytes)
	owner, name := strings.Repeat("o", 39), strings.Repeat("n", 100)
	repository := "https://api.github.com/repos/" + owner + "/" + name
	sha := strings.Repeat("0", 40)
	contentsURL := repository + "/contents/" + path + "?ref=" + sha
	blobURL := repository + "/git/blobs/" + sha
	pageURL := "https://github.com/" + owner + "/" + name + "/blob/" + sha + "/" + path
	answer, err := json.Marshal(contentsAnswer{
		Name: path, Path: path, SHA: sha, Size: limits.MaxSourceBytes,
		URL: contentsURL, HTMLURL: pageURL, BlobURL: blobURL,
		DownloadURL: "https://raw.githubusercontent.com/" + owner + "/" + name + "/" + sha + "/" + path,
		Type:        "file", Content: strings.Join(lines, "\n") + "\n", Encoding: "base64",
		Links: contentsLinks{Self: contentsURL, Blob: blobURL, HTML: pageURL},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(answer) > github.MaxResponseBytes() {
		t.Errorf("the largest contents answer is %d bytes, more than the %d-byte bound", len(answer), github.MaxResponseBytes())
	}
}

func TestAnAnswerAtTheResponseBoundIsReadAndOneByteMoreIsRefused(t *testing.T) {
	bound := github.MaxResponseBytes()
	for form, padding := range map[string]int{"at the bound": bound - len("[]"), "one over": bound - len("[]") + 1} {
		t.Run(form, func(t *testing.T) {
			server := serving(t, "/repos/acme/payments/pulls/412/files", "["+strings.Repeat(" ", padding)+"]")
			settings := settingsAt(server)
			settings.MaxResponseBytes = bound

			_, err := sourceWith(t, settings).FilesChanged(context.Background(), locatorOf(t, "acme/payments", 412))

			if isOver := padding > bound-len("[]"); isOver != errors.Is(err, github.ErrResponseTooLarge) {
				t.Errorf("an answer %s: error is %v", form, err)
			}
		})
	}
}

type contentsAnswer struct {
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	SHA         string        `json:"sha"`
	Size        int           `json:"size"`
	URL         string        `json:"url"`
	HTMLURL     string        `json:"html_url"`
	BlobURL     string        `json:"git_url"`
	DownloadURL string        `json:"download_url"`
	Type        string        `json:"type"`
	Content     string        `json:"content"`
	Encoding    string        `json:"encoding"`
	Links       contentsLinks `json:"_links"`
}

type contentsLinks struct {
	Self string `json:"self"`
	Blob string `json:"git"`
	HTML string `json:"html"`
}
