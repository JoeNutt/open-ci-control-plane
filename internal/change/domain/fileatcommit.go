// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "fmt"

type FileAtCommit struct {
	repository GovernedRepository
	commit     CommitSHA
	path       RepositoryPath
}

func ParseFileAtCommit(repository GovernedRepository, commit CommitSHA, path string) (FileAtCommit, error) {
	if repository == (GovernedRepository{}) {
		return FileAtCommit{}, fmt.Errorf("%w: none stated", ErrMalformedRepository)
	}
	if commit == (CommitSHA{}) {
		return FileAtCommit{}, fmt.Errorf("%w: none stated", ErrMalformedCommitSHA)
	}
	parsed, err := ParseRepositoryPath(path)
	if err != nil {
		return FileAtCommit{}, fmt.Errorf("%q: %w", path, err)
	}
	return FileAtCommit{repository: repository, commit: commit, path: parsed}, nil
}

func (file FileAtCommit) Repository() GovernedRepository { return file.repository }

func (file FileAtCommit) Commit() CommitSHA { return file.commit }

func (file FileAtCommit) Path() RepositoryPath { return file.path }

func (file FileAtCommit) String() string {
	return fmt.Sprintf("%s in %s at %s", file.path, file.repository, file.commit)
}
