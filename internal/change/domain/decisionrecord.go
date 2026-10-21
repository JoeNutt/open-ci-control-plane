// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "encoding/json"

type DecisionRecord struct {
	decisionID DecisionID
	revision   BundleRevision
}

func RecordFor(decided FamilyDecision, id DecisionID) DecisionRecord {
	return DecisionRecord{decisionID: id, revision: decided.Decision.Revision()}
}

func (record DecisionRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(publishedRecord{
		DecisionID:     record.decisionID.String(),
		BundleRevision: publishedRevisionOf(record.revision),
	})
}

type publishedRecord struct {
	DecisionID     string  `json:"decisionId"`
	BundleRevision *string `json:"bundleRevision,omitempty"`
}

func publishedRevisionOf(revision BundleRevision) *string {
	if !revision.isKnown() {
		return nil
	}
	stated := revision.String()
	return &stated
}
