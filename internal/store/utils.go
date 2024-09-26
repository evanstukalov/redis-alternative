package store

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func reGroupOne(keyStream string, id string, store *Store) (string, error) {
	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Debug("Matches group")

	lastStreamId, err := store.GetLastStreamID(keyStream, id)
	if err != nil {
		return lastStreamId, nil
	}

	logrus.Debug("lastStreamId ", lastStreamId)

	err = compareIDs(id, lastStreamId)
	if err != nil {
		return "", err
	}

	return id, nil
}

func reGroupTwo(keyStream string, id string, store *Store) (string, error) {
	logrus.WithField("id", id).Debug("Matches group any sequence")

	newTimestamp, _ := splitID(id)

	defaultValue := fmt.Sprintf("%s-0", newTimestamp)

	if newTimestamp == "0" {
		defaultValue = "0-1"
	}

	lastStreamId, err := store.GetLastStreamID(keyStream, defaultValue)
	if err != nil {
		return lastStreamId, nil
	}

	lastTimestamp, _ := splitID(lastStreamId)

	if newTimestamp != lastTimestamp {
		id, err = store.CreateNewStreamID(keyStream, id)
		if err != nil {
			return "", err
		}
	} else {
		id, err = store.IncrStreamID(keyStream)
		if err != nil {
			return "", err
		}
	}

	logrus.Debug("result ID ", id)

	return id, nil
}

func reGroupThree(id string) (string, error) {
	logrus.WithField("id", id).Debug("Matches group any")

	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	id = fmt.Sprintf("%d-%d", timestamp, 0)

	return id, nil
}

func FormID(keyStream string, id string, store *Store) (string, error) {
	logrus.Debug(keyStream, id)
	logrus.Debug(store.store[keyStream].GetStorable())

	reGroup := regexp.MustCompile(`^\d+-\d+$`)
	reGroupAnySequence := regexp.MustCompile(`^\d+-\*$`)
	reGroupAny := regexp.MustCompile(`^\*$`)

	switch {
	case reGroup.MatchString(id):
		return reGroupOne(keyStream, id, store)

	case reGroupAnySequence.MatchString(id):
		return reGroupTwo(keyStream, id, store)

	case reGroupAny.MatchString(id):
		return reGroupThree(id)
		// generate timestamp + lastvalue+1
	}

	logrus.Info("No match")

	return "", errors.New("ID format is not valid")
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

func binarySearch(messages StreamMessages, targetID string) int {
	return sort.Search(len(messages.Messages), func(i int) bool {
		return messages.Messages[i].ID >= targetID
	})
}

func GetRangedMessages(messages StreamMessages, i int, i2 int) []StreamMessage {
	return messages.Messages[i : i2+1]
}
