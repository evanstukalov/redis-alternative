package commands

import (
	"context"
	"io"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
)

type Command interface {
	Execute(ctx context.Context, conn io.Writer, config interfaces.IConfig, args []string)
}

type ITransactionBuffer interface {
	Start()
	Discard()
	IsActive() bool
	IsBufferEmpty() bool
	Unactivate()
	PutCommand(command *BufferedCommand)
	PopCommands() []*BufferedCommand
}

type ITransactions interface {
	GetTransactionBuffer(conn net.Conn) *TransactionBuffer
	AddConnection(conn net.Conn)
}
