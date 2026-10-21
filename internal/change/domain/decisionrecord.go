// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type DecisionRecord struct {
	decisionID DecisionID
	revision   BundleRevision
	outcome    Outcome
	verdict    Verdict
	masked     MaskedInput
}

func RecordFor(decided FamilyDecision, document Document, id DecisionID) (DecisionRecord, error) {
	masked, err := MaskedInputOf(document)
	if err != nil {
		return DecisionRecord{}, err
	}
	return DecisionRecord{
		decisionID: id,
		revision:   decided.Decision.Revision(),
		outcome:    decided.Decision.Outcome(),
		verdict:    decided.Verdict(),
		masked:     masked,
	}, nil
}

func (record DecisionRecord) DecisionID() DecisionID   { return record.decisionID }
func (record DecisionRecord) Revision() BundleRevision { return record.revision }
func (record DecisionRecord) Outcome() Outcome         { return record.outcome }
func (record DecisionRecord) Verdict() Verdict         { return record.verdict }
func (record DecisionRecord) MaskedInput() MaskedInput { return record.masked }
