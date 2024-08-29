package store

import (
	"errors"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type ValueWithExpiration struct {
	value     string
	ExpiredAt *time.Time
}

type Store struct {
	store map[string]ValueWithExpiration
	mutex sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		store: make(map[string]ValueWithExpiration),
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

	s.store[key] = ValueWithExpiration{
		value:     value,
		ExpiredAt: expirationTime,
	}

	log.Println("Set handler: ", key, value)
}

func (s *Store) Get(key string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if value, ok := s.store[key]; !ok {
		return "", errors.New("key does not exists")
	} else {
		log.Println("Get handler: ", key, value.value)
		return value.value, nil
	}
}

func (s *Store) Incr(key string) (int, error) {
	log.WithFields(log.Fields{"key": key}).Info("Incrementing key in store")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	v, ok := s.store[key]
	if !ok {
		s.store[key] = ValueWithExpiration{
			value: "1",
		}
		return 1, nil
	}

	intValue, err := strconv.Atoi(v.value)
	if err != nil {
		return 0, errors.New("Unsupported type")
	}

	intValue++
	s.store[key] = ValueWithExpiration{
		value: strconv.Itoa(intValue),
	}
	return intValue, nil
}

func (s *Store) Remove(key string) {
	delete(s.store, key)
	log.WithField("key", key).Info("Removing key from store")
}
