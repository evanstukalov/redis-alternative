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

type ValueWithType struct {
	Value    string
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
