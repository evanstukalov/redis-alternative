package interfaces

type IConfig interface {
	GetPort() int
	GetRole() string
	GetRedisDir() string
	GetRedisDbFileName() string
	GetMaster() IMaster
	GetSlave() ISlave

	SetPort(port int)
	SetRole(role string)
	SetRedisDir(dir string)
	SetRedisDbFileName(dbFileName string)
	SetMaster(master IMaster)
	SetSlave(slave ISlave)

	Initialize(flags IFlags)
}

type ISlave interface {
	AddOffset(offset int64)
	GetReplicaOf() string
	GetOffset() int64
}

type IMaster interface {
	AddMasterReplOffset(masterReplOffset int64)
	GetMasterReplId() string
	GetMasterReplOffset() int64
}

type IFlags interface {
	GetPort() int
	GetReplicaof() string
	GetDir() string
	GetDbFileName() string
}
