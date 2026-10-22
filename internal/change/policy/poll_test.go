// SPDX-License-Identifier: GPL-3.0-or-later

package policy_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JoeNutt/open-ci-control-plane/internal/change/domain"
	"github.com/JoeNutt/open-ci-control-plane/internal/change/policy"
)

var errForgeUnreachable = errors.New("the forge is unreachable")

func TestPollAtEachTickKeepsTheActiveRevisionThroughAFailedPollAndLogsItUntilAValidOne(t *testing.T) {
	var logged bytes.Buffer
	active := activePolicyLoggingTo(t, slog.NewJSONHandler(&logged, nil))
	recorded := policyRepositoryAt(t, firstPolicyCommit, liveFiles(t))
	recordCommit(t, recorded, thirdPolicyCommit, liveFiles(t))
	ports := portsOver(recorded)
	ports.Heads = &scriptedHeads{answers: []headAnswer{
		{head: policyCommit(t, firstPolicyCommit)},
		{failure: errForgeUnreachable},
		{head: policyCommit(t, thirdPolicyCommit)},
	}}
	ticks := make(chan time.Time)
	polled := pollInBackground(readerWith(t, ports, active), ticks)

	ticks <- time.Time{}
	ticks <- time.Time{}
	afterFailure := acceptedRevisionOf(t, active)
	close(ticks)
	<-polled

	if afterFailure != policyCommit(t, firstPolicyCommit) {
		t.Errorf("after a failed poll the active revision is %s, want %s", afterFailure, firstPolicyCommit)
	}
	if revision := acceptedRevisionOf(t, active); revision != policyCommit(t, thirdPolicyCommit) {
		t.Errorf("after a valid poll the active revision is %s, want %s", revision, thirdPolicyCommit)
	}
	if !strings.Contains(logged.String(), errForgeUnreachable.Error()) {
		t.Errorf("the failed poll was not logged: %q", logged.String())
	}
}

func pollInBackground(reader *policy.Reader, ticks <-chan time.Time) <-chan struct{} {
	polled := make(chan struct{})
	go func() {
		defer close(polled)
		reader.PollAtEachTick(context.Background(), ticks)
	}()
	return polled
}

type headAnswer struct {
	head    domain.CommitSHA
	failure error
}

type scriptedHeads struct {
	mutex   sync.Mutex
	answers []headAnswer
}

func (scripted *scriptedHeads) DefaultBranchHeadOf(context.Context, domain.PolicyRepository) (domain.CommitSHA, error) {
	scripted.mutex.Lock()
	defer scripted.mutex.Unlock()

	answer := scripted.answers[0]
	if len(scripted.answers) > 1 {
		scripted.answers = scripted.answers[1:]
	}
	return answer.head, answer.failure
}
