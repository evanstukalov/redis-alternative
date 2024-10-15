package config

import (
	"sync/atomic"
)

type Slave struct {
	Replicaof string
	Offset    atomic.Int64
}

func (s *Slave) GetReplicaOf() string {
	return s.Replicaof
}

func (s *Slave) GetOffset() int64 {
	return s.Offset.Load()
}

func (s *Slave) AddOffset(offset int64) {
	s.Offset.Add(offset)
}
