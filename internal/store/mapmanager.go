package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func NewStore() *Store {
	logrus.Info("Creating new store")
	return &Store{
		store: make(map[string]Value),
	}
}

func (s *Store) Set(key string, value string, px *int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var expirationTime *time.Time
	if px != nil {
		t := time.Now().Add(time.Duration(*px) * time.Millisecond)

		expirationTime = &t
	}

	s.store[key] = Value{
		ValueData: ValueWithType{Data: StringT(value), DataType: StringType},
		ExpiredAt: expirationTime,
	}

	log.Println("Set handler: ", key, value)
}

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

func (s *Store) GetLastStreamID(keyStream string, defaultValue string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	value, ok := s.store[keyStream]
	if !ok {
		return defaultValue, errors.New("key does not exists")
	}

	logrus.Error(value.GetStorable())

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

func (s *Store) Get(key string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if value, ok := s.store[key]; !ok {
		return "", errors.New("key does not exists")
	} else {
		log.Println("Get handler: ", key, value.ValueData)
		if str, ok := value.ValueData.Data.(StringT); ok {
			return string(str), nil
		}

		return "", errors.New("Value is not of type StringT")
	}
}

func (s *Store) GetType(key string) (Datatype, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if value, ok := s.store[key]; !ok {
		return "", errors.New("key does not exists")
	} else {
		return value.ValueData.DataType, nil
	}
}

func (s *Store) Incr(key string) (int, error) {
	log.WithFields(log.Fields{"key": key}).Info("Incrementing key in store")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.store[key]
	if !ok {
		s.store[key] = Value{
			ValueWithType{Data: StringT("1"), DataType: StringType},
			nil,
		}
		return 1, nil
	}

	intValue, err := strconv.Atoi(string(v.ValueData.Data.(StringT)))
	if err != nil {
		return 0, errors.New("Unsupported type")
	}

	intValue++
	s.store[key] = Value{
		ValueData: ValueWithType{Data: StringT(strconv.Itoa(intValue))},
	}
	return intValue, nil
}

func (s *Store) Remove(key string) {
	delete(s.store, key)
	log.WithField("key", key).Info("Removing key from store")
}
