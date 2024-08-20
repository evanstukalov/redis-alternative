package config

import "sync/atomic"

type Config struct {
	Port   int
	Role   string
	Master *Master
	Slave  *Slave

	RedisDir        string
	RedisDbFileName string
}

type Slave struct {
	Replicaof string
	Offset    atomic.Int64
}

type Master struct {
	MasterReplId     string
	MasterReplOffset atomic.Int64
}
