// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "encoding/json"

type DecisionRecord struct {
	decisionID DecisionID
}

func RecordFor(decided FamilyDecision, document Document, id DecisionID) (DecisionRecord, error) {
	return DecisionRecord{decisionID: id}, nil
}

func (record DecisionRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DecisionID string `json:"decisionId"`
	}{record.decisionID.String()})
}
