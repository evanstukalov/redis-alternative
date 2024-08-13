package config

import "sync/atomic"

type Config struct {
	Port   int
	Role   string
	Master *Master
	Slave  *Slave
}

type Slave struct {
	Replicaof string
	Offset    atomic.Int64
}

type Master struct {
	MasterReplId     string
	MasterReplOffset int
}
