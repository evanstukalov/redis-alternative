package config

import (
	"sync/atomic"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
)

type Master struct {
	MasterReplId     string
	MasterReplOffset atomic.Int64
}

func (m *Master) SetMaster(master interfaces.IMaster) {
	m.MasterReplId = master.GetMasterReplId()
	m.MasterReplOffset.Store(master.GetMasterReplOffset())
}

func (m *Master) AddMasterReplOffset(masterReplOffset int64) {
	m.MasterReplOffset.Add(masterReplOffset)
}

func (m *Master) GetMasterReplId() string {
	return m.MasterReplId
}

func (m *Master) GetMasterReplOffset() int64 {
	return m.MasterReplOffset.Load()
}
