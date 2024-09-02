package transactions

import (
	"context"
	"net"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
)

type Command interface {
	Execute(ctx context.Context, conn net.Conn, config config.Config, args []string)
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

func NewTransactionBuffer() *TransactionBuffer {
	logrus.Info("Creating new transaction buffer")

	return &TransactionBuffer{
		CommandsBuffer: make([]*BufferedCommand, 8),
	}
}

func (t *TransactionBuffer) StartTransaction() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Active = true
}

func (t *TransactionBuffer) IsTransactionActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Active
}

func (t *TransactionBuffer) EndTransaction() {
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

	t.CommandsBuffer = make([]*BufferedCommand, 8)

	return result
}
