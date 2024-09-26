package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (s *Store) XAdd(key string, streamValue StreamMessage) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, exists := s.store[key]
	if !exists {
		s.store[key] = Value{
			ValueData: ValueWithType{
				Data:     StreamMessages{Messages: []StreamMessage{streamValue}},
				DataType: StreamType,
			},
		}
		return nil
	}

	streamMessages := value.ValueData.Data.(StreamMessages)
	streamMessages.Messages = append(streamMessages.Messages, streamValue)

	value.ValueData.Data = streamMessages

	s.store[key] = value

	return nil
}

func (s *Store) GetStream(
	key string,
	rangeTargets [2]string,
) ([]StreamMessage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if value, ok := s.store[key]; !ok {
		return []StreamMessage{}, errors.New("key does not exists")
	} else {

		index := binarySearch(value.GetStorable().(StreamMessages), rangeTargets[0])
		indexTwo := binarySearch(value.GetStorable().(StreamMessages), rangeTargets[1])

		logrus.Info("range targets", rangeTargets[0], rangeTargets[1])
		logrus.Info("targets indexes", index, indexTwo)

		return GetRangedMessages(value.GetStorable().(StreamMessages), index, indexTwo), nil

	}
}

func (s *Store) GetLastStreamID(keyStream string, defaultValue string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	value, ok := s.store[keyStream]
	if !ok {
		return defaultValue, errors.New("key does not exists")
	}

	id := value.GetStorable().(StreamMessages).Messages[len(value.GetStorable().(StreamMessages).Messages)-1].ID

	return id, nil
}

func (s *Store) IncrStreamID(keyStream string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, ok := s.store[keyStream]
	if !ok {
		return "0-1", errors.New("key does not exists")
	}

	id := value.GetStorable().(StreamMessages).Messages[len(value.GetStorable().(StreamMessages).Messages)-1].ID

	parts := strings.Split(id, "-")
	lastValue, _ := strconv.Atoi(parts[1])
	newID := fmt.Sprintf("%s-%d", parts[0], lastValue+1)

	return newID, nil
}

func (s *Store) CreateNewStreamID(keyStream string, id string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, ok := s.store[keyStream]
	if !ok {
		return "0-1", errors.New("key does not exists")
	}

	parts := strings.Split(id, "-")
	newID := fmt.Sprintf("%s-%s", parts[0], "0")

	return newID, nil
}
