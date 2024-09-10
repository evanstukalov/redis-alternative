package store

import (
	"sync"
	"time"
)

type Datatype string

const (
	StringType Datatype = "string"
	StreamType Datatype = "stream"
)

type Storable interface {
	IsStorable()
}

type StringT string

func (s StringT) IsStorable() {}

type StreamMessages struct {
	Messages []StreamMessage
}

type StreamMessage struct {
	ID     string
	Fields map[string]interface{}
}

func (s StreamMessages) IsStorable() {}

type ValueWithType struct {
	Value    Storable
	DataType Datatype
}

type Value struct {
	Value     ValueWithType
	ExpiredAt *time.Time
}

type Store struct {
	store map[string]Value
	mutex sync.RWMutex
}
