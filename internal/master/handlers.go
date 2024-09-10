package master

import (
	"context"
	"io"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/transactions"
)

type CommandHandler interface {
	SetNext(handler CommandHandler) CommandHandler
	Handle(
		ctx context.Context,
		conn io.Writer,
		config config.Config,
		args []string,
		cmd commands.Command,
	) bool
}

type BaseCommandHandler struct {
	next CommandHandler
}

func (b *BaseCommandHandler) SetNext(handler CommandHandler) CommandHandler {
	b.next = handler
	return handler
}

func (b *BaseCommandHandler) Handle(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
	cmd commands.Command,
) bool {
	result := b.HandleNext(ctx, conn, config, args, cmd)

	return result
}

func (b *BaseCommandHandler) HandleNext(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
	cmd commands.Command,
) bool {
	if b.next != nil {
		return b.next.Handle(ctx, conn, config, args, cmd)
	}
	return true
}

type DiscardConditionHandler struct {
	BaseCommandHandler
}

func (b *DiscardConditionHandler) Handle(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
	cmd commands.Command,
) bool {
	if _, ok := cmd.(*commands.DiscardCommand); ok {
		cmd.Execute(ctx, conn, config, args)
		return false
	}

	return b.HandleNext(ctx, conn, config, args, cmd)
}

type QueuedConditionHandler struct {
	BaseCommandHandler
}

func (b *QueuedConditionHandler) Handle(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
	cmd commands.Command,
) bool {
	transactionsObj := transactions.GetTransactionsObj(ctx)
	transactionBufferObj := transactionsObj.GetTransactionBuffer(conn.(net.Conn))

	if _, ok := cmd.(*commands.ExecCommand); !ok && transactionBufferObj.IsTransactionActive() {

		transactionBufferObj.PutCommand(&transactions.BufferedCommand{
			CMD:  cmd,
			Args: args,
		})

		conn.Write([]byte("+QUEUED\r\n"))
		return false
	}

	return b.HandleNext(ctx, conn, config, args, cmd)
}
