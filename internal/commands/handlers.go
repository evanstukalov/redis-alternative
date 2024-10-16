package commands

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

func (c *ConfigCommand) handleGet(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	commands := map[string]CommandHandler{
		"dir":        c.handleGetDir,
		"dbfilename": c.handleGetDbFile,
	}

	if handler, exists := commands[args[2]]; exists {
		handler(ctx, conn, config, args)
	}
}

func (c *ConfigCommand) handleGetDir(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	dir := fmt.Sprintf(
		"*2\r\n$3\r\ndir\r\n$%d\r\n%s\r\n",
		len(config.GetRedisDir()),
		config.GetRedisDir(),
	)
	conn.Write([]byte(dir))
}

func (c *ConfigCommand) handleGetDbFile(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	dir := fmt.Sprintf(
		"*2\r\n$9\r\ndbfilename\r\n$%d\r\n%s\r\n",
		len(config.GetRedisDbFileName()),
		config.GetRedisDbFileName(),
	)
	conn.Write([]byte(dir))
}

func (c *KeysCommand) handleAll(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	fileContent := redis.ReadFile(config.GetRedisDir() + "/" + config.GetRedisDbFileName())
	conn.Write([]byte(fmt.Sprintf("*1\r\n$%d\r\n%s\r\n", len(fileContent), fileContent)))
}

func (c *ReplConfCommand) handleMaster(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
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
	config interfaces.IConfig,
	args []string,
) {
	if args[1] == "GETACK" && args[2] == "*" {
		offset := config.GetSlave().GetOffset()
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
	config interfaces.IConfig,
	args []string,
) {
	conn.Write([]byte("+OK\r\n"))
}

func (c *ReplConfCommand) handleAck(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	clientsObj, ok := utils.GetFromCtx[*clients.Clients](ctx, "clients")

	if !ok {
		logrus.Error("No store in context")
		return
	}

	if conn, ok := conn.(net.Conn); ok {

		_, ok := clientsObj.Clients[conn]

		if ok {
			offset, _ := strconv.Atoi(args[2])
			clientsObj.SetOffset(conn, offset)
		}
	}
}
