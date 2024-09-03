package commands

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

func (c *ReplConfCommand) handleMaster(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	commands := map[string]CommandHandler{
		"ACK":            c.handleAck,
		"capa":           c.handleOk,
		"listening-port": c.handleOk,
	}

	if handler, exists := commands[args[1]]; exists {
		handler(ctx, conn, config, args)
	}
}

func (c *ReplConfCommand) handleSlave(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	if args[1] == "GETACK" && args[2] == "*" {
		offset := config.Slave.Offset.Load()
		byteCount := len(strconv.Itoa(int(offset)))
		conn.Write(
			[]byte(
				fmt.Sprintf(
					"*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$%d\r\n%d\r\n",
					byteCount,
					offset,
				),
			),
		)
	}
}

func (c *ReplConfCommand) handleOk(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	conn.Write([]byte("+OK\r\n"))
}

func (c *ReplConfCommand) handleAck(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	clients := utils.GetClientsObj(ctx)

	if conn, ok := conn.(net.Conn); ok {

		_, ok := clients.Clients[conn]

		if ok {
			offset, _ := strconv.Atoi(args[2])
			clients.SetOffset(conn, offset)
		}
	}
}
