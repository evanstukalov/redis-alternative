package store

import (
	"errors"
	"strings"

	"github.com/sirupsen/logrus"
)

func FormID(keyStream string, id string, store *Store) (string, error) {
	logrus.Debug(keyStream, id)

	lastStreamId, err := store.GetLastStreamID(keyStream)
	if err != nil {
		return lastStreamId, nil
	}

	if id == "*" {
		// generate timestamp + lastvalue+1
	}

	p1, p2 := splitID(id)

	if p2 == "*" {
		p1, p2 = splitID(lastStreamId)
	}

	err = compareIDs(id, lastStreamId)
	if err != nil {
		return "", err
	}

	res := p1 + "-" + p2
	logrus.Debug(res)
	return res, nil
}

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
