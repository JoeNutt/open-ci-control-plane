// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"context"
	"time"
)

const PollInterval = 60 * time.Second

const (
	failedPollMessage = "policy poll failed"
	repositoryLogKey  = "repository"
)

func (reader *Reader) PollAtEachTick(ctx context.Context, ticks <-chan time.Time) {
	for {
		reader.readRevisionLoggingFailure(ctx)
		select {
		case <-ctx.Done():
			return
		case _, isOpen := <-ticks:
			if !isOpen {
				return
			}
		}
	}
}

func (reader *Reader) readRevisionLoggingFailure(ctx context.Context) {
	if err := reader.ReadRevision(ctx); err != nil {
		reader.active.log.Error(failedPollMessage,
			repositoryLogKey, reader.repository.String(), reasonLogKey, err.Error())
	}
}
