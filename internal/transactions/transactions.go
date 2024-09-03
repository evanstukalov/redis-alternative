package transactions

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
)

type Command interface {
	Execute(ctx context.Context, conn io.Writer, config config.Config, args []string)
}

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
	logrus.Info("Creating new transaction buffer")

	return &TransactionBuffer{
		CommandsBuffer: make([]*BufferedCommand, 0, 8),
	}
}

func NewTransaction() *Transactions {
	logrus.Info("Creating new transactions obj")

	return &Transactions{
		Values: make(map[net.Conn]*TransactionBuffer),
	}
}

func (t *Transactions) AddConnection(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Values[conn] = NewTransactionBuffer()
}

func (t *TransactionBuffer) StartTransaction() {
	logrus.Info("Starting transaction")
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Active = true
}

func (t *TransactionBuffer) IsTransactionActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := t.Active

	logrus.WithFields(logrus.Fields{
		"package":  "transactions",
		"function": "IsTransactionActive",
		"result":   result,
	}).Info()

	return result
}

func (t *TransactionBuffer) IsBufferEmpty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := len(t.CommandsBuffer) == 0

	logrus.WithFields(logrus.Fields{
		"package":  "transactions",
		"function": "IsBufferEmpty",
		"result":   result,
	})

	return result
}

func (t *TransactionBuffer) EndTransaction() {
	logrus.Info("Ending transaction")
	t.mu.Lock()
	defer t.mu.Unlock()
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
