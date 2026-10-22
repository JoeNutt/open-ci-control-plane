// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrDocumentTooLarge = errors.New("change document exceeds the document byte bound")

func ParseBoundedDocument(document Document, limits Limits) (Document, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return Document{}, fmt.Errorf("measure the change document: %w", err)
	}
	if len(encoded) > limits.MaxDocumentBytes {
		return Document{}, fmt.Errorf("%w: %d exceeds %d", ErrDocumentTooLarge, len(encoded), limits.MaxDocumentBytes)
	}
	return document, nil
}
