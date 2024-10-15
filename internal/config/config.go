package config

import (
	"github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
)

type Config struct {
	Port   int
	Role   string
	Master *Master
	Slave  *Slave

	RedisDir        string
	RedisDbFileName string
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Initialize(flags interfaces.IFlags) {
	c.SetPort(flags.GetPort())
	c.SetRedisDir(flags.GetDir())
	c.SetRedisDbFileName(flags.GetDbFileName())

	if flags.GetReplicaof() == "" {
		c.SetRole("master")
		c.SetMaster(&Master{
			MasterReplId: "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
		})
		logrus.Info("Master is Initialized")
	} else {
		c.SetRole("slave")
		c.SetSlave(&Slave{
			Replicaof: flags.GetReplicaof(),
		})
	}
}

func (c *Config) GetPort() int {
	return c.Port
}

func (c *Config) GetRole() string {
	return c.Role
}

func (c *Config) GetMaster() interfaces.IMaster {
	return c.Master
}

func (c *Config) GetSlave() interfaces.ISlave {
	return c.Slave
}

func (c *Config) GetRedisDir() string {
	return c.RedisDir
}

func (c *Config) GetRedisDbFileName() string {
	return c.RedisDbFileName
}

func (c *Config) SetPort(port int) {
	c.Port = port
}

func (c *Config) SetRole(role string) {
	c.Role = role
}

func (c *Config) SetRedisDir(dir string) {
	c.RedisDir = dir
}

func (c *Config) SetRedisDbFileName(dbFileName string) {
	c.RedisDbFileName = dbFileName
}

func (c *Config) SetMaster(master interfaces.IMaster) {
	if m, ok := master.(*Master); ok {
		c.Master = m
		return
	}
	panic("Master is not initialized")
}

func (c *Config) SetSlave(slave interfaces.ISlave) {
	if s, ok := slave.(*Slave); ok {
		c.Slave = s
		return
	}
	panic("Slave is not initialized")
}
