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
	Data     Storable
	DataType Datatype
}

type Value struct {
	ValueData ValueWithType
	ExpiredAt *time.Time
}

func (v Value) GetStorable() Storable {
	return v.ValueData.Data
}

type Store struct {
	store map[string]Value
	mutex sync.RWMutex
}
