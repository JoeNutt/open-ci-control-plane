// SPDX-License-Identifier: GPL-3.0-or-later

package policy

import (
	"context"
	"time"
)

func (reader *Reader) PollAtEachTick(ctx context.Context, ticks <-chan time.Time) {
	for {
		if err := reader.ReadRevision(ctx); err != nil {
			reader.active.log.Error("policy poll failed", "reason", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case _, open := <-ticks:
			if !open {
				return
			}
		}
	}
}
