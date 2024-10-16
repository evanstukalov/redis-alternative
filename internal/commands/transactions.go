package commands

import (
	"net"
	"sync"
)

type BufferedCommand struct {
	CMD  Command
	Args []string
}

type TransactionBuffer struct {
	CommandsBuffer []*BufferedCommand
	Active         bool
	mu             sync.Mutex
}

type Transactions struct {
	Values map[net.Conn]*TransactionBuffer
	mu     sync.Mutex
}

func NewTransactionBuffer() *TransactionBuffer {
	return &TransactionBuffer{
		CommandsBuffer: make([]*BufferedCommand, 0, 8),
	}
}

func NewTransaction() *Transactions {
	return &Transactions{
		Values: make(map[net.Conn]*TransactionBuffer),
	}
}

func (t *Transactions) AddConnection(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Values[conn] = NewTransactionBuffer()
}

func (t *Transactions) GetTransactionBuffer(conn net.Conn) *TransactionBuffer {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Values[conn]
}

func (t *TransactionBuffer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Active = true
}

func (t *TransactionBuffer) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := t.Active

	return result
}

func (t *TransactionBuffer) IsBufferEmpty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := len(t.CommandsBuffer) == 0

	return result
}

func (t *TransactionBuffer) UnActivate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Active = false
}

func (t *TransactionBuffer) Discard() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CommandsBuffer = make([]*BufferedCommand, 0, 8)
	t.Active = false
}

func (t *TransactionBuffer) PutCommand(command *BufferedCommand) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CommandsBuffer = append(t.CommandsBuffer, command)
}

func (t *TransactionBuffer) PopCommands() []*BufferedCommand {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]*BufferedCommand, len(t.CommandsBuffer))
	copy(result, t.CommandsBuffer)

	t.CommandsBuffer = make([]*BufferedCommand, 0, 8)

	return result
}
