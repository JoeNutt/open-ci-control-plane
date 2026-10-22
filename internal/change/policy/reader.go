// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"context"
	"fmt"

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
	head, err := reader.headCommit(ctx)
	if err != nil {
		return fmt.Errorf("read the head of %s: %w", reader.repository, err)
	}
	listing, err := reader.policyListingAt(ctx, head)
	if err != nil {
		return fmt.Errorf("list the policy files of %s at %s: %w", reader.repository, head.Commit(), err)
	}
	files, err := reader.policyFilesAt(ctx, head, listing)
	if err != nil {
		return fmt.Errorf("read the policy files of %s at %s: %w", reader.repository, head.Commit(), err)
	}
	return reader.active.Offer(head.Commit(), files)
}

func (reader *Reader) headCommit(ctx context.Context) (domain.PolicyCommit, error) {
	head, err := reader.ports.Heads.DefaultBranchHeadOf(ctx, reader.repository)
	if err != nil {
		return domain.PolicyCommit{}, err
	}
	return domain.ParsePolicyCommit(reader.repository, head)
}

func (reader *Reader) policyListingAt(ctx context.Context, at domain.PolicyCommit) (domain.PolicyListing, error) {
	listed, err := reader.ports.Listings.ListedFilesAt(ctx, at)
	if err != nil {
		return domain.PolicyListing{}, err
	}
	return domain.ParsePolicyListing(placedFilesIn(listed))
}

func placedFilesIn(listed []domain.ListedFile) []domain.ListedFile {
	placed := make([]domain.ListedFile, 0, len(listed))
	for _, file := range listed {
		if isPlacedPolicyFile(file.Path) {
			placed = append(placed, file)
		}
	}
	return placed
}

func (reader *Reader) policyFilesAt(
	ctx context.Context, at domain.PolicyCommit, listing domain.PolicyListing,
) ([]domain.PolicyFile, error) {
	files := make([]domain.PolicyFile, 0, len(listing.Files()))
	for _, listed := range listing.Files() {
		path, err := domain.ParseRepositoryPath(listed.Path)
		if err != nil {
			return nil, err
		}
		content, err := reader.ports.Contents.ContentsAt(ctx, at, path)
		if err != nil {
			return nil, err
		}
		files = append(files, domain.PolicyFile{Path: listed.Path, Content: content})
	}
	return files, nil
}
