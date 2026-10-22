// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"context"
	"strings"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
)

type ReaderPorts struct {
	Heads    domain.DefaultBranchHeads
	Listings domain.PolicyListings
	Contents domain.PolicyContents
}

type Reader struct {
	repository domain.PolicyRepository
	ports      ReaderPorts
	active     *ActivePolicy
}

func ParseReader(repository domain.PolicyRepository, ports ReaderPorts, active *ActivePolicy) (*Reader, error) {
	return &Reader{repository: repository, ports: ports, active: active}, nil
}

func (reader *Reader) ReadRevision(ctx context.Context) error {
	head, err := reader.ports.Heads.DefaultBranchHeadOf(ctx, reader.repository)
	if err != nil {
		return err
	}
	at, err := domain.ParsePolicyCommit(reader.repository, head)
	if err != nil {
		return err
	}
	listed, err := reader.ports.Listings.ListedFilesAt(ctx, at)
	if err != nil {
		return err
	}
	var placed []domain.ListedFile
	for _, file := range listed {
		if strings.HasPrefix(file.Path, catalogDirectory) || strings.HasPrefix(file.Path, bindingsDirectory) {
			placed = append(placed, file)
		}
	}
	listing, err := domain.ParsePolicyListing(placed)
	if err != nil {
		return err
	}
	var files []domain.PolicyFile
	for _, file := range listing.Files() {
		path, err := domain.ParseRepositoryPath(file.Path)
		if err != nil {
			return err
		}
		content, err := reader.ports.Contents.ContentsAt(ctx, at, path)
		if err != nil {
			return err
		}
		files = append(files, domain.PolicyFile{Path: file.Path, Content: content})
	}
	return reader.active.Offer(head, files)
}
