package config

type Flags struct {
	Port       int
	ReplicaOf  string
	Dir        string
	DbFileName string
}

func (f *Flags) GetPort() int {
	return f.Port
}

func (f *Flags) GetReplicaof() string {
	return f.ReplicaOf
}

func (f *Flags) GetDir() string {
	return f.Dir
}

func (f *Flags) GetDbFileName() string {
	return f.DbFileName
}
