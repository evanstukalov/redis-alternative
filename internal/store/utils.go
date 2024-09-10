package store

import (
	"errors"
	"strings"
)

func splitID(id string) (string, string) {
	parts := strings.Split(id, "-")
	return parts[0], parts[1]
}

func compareIDs(id1 string, id2 string) error {
	millisecondPart1, sequencePart1 := splitID(id1)
	millisecondPart2, sequencePart2 := splitID(id2)

	if id1 == "0-0" {
		return errors.New("The ID specified in XADD must be greater than 0-0")
	}

	if isIDSmallerOrEqual(millisecondPart1, millisecondPart2, sequencePart1, sequencePart2) {
		return errors.New(
			"The ID specified in XADD is equal or smaller than the target stream top item",
		)
	}

	return nil
}

func isIDSmallerOrEqual(ms1, ms2, seq1, seq2 string) bool {
	return ms1 < ms2 || (ms1 == ms2 && seq1 <= seq2)
}
