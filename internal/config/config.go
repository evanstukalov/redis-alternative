package config

type Config struct {
	Port             int
	Role             string
	MasterReplId     string
	MasterReplOffset int
	Replicaof        string
}
