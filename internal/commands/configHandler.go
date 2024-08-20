package commands

import (
	"context"
	"fmt"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/config"
)

func (c *ConfigCommand) handleGet(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
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
	conn net.Conn,
	config config.Config,
	args []string,
) {
	dir := fmt.Sprintf(
		"*2\r\n$3\r\ndir\r\n$%d\r\n%s\r\n",
		len(config.RedisDir),
		config.RedisDir,
	)
	conn.Write([]byte(dir))
}

func (c *ConfigCommand) handleGetDbFile(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	dir := fmt.Sprintf(
		"*2\r\n$9\r\ndbfilename\r\n$%d\r\n%s\r\n",
		len(config.RedisDbFileName),
		config.RedisDbFileName,
	)
	conn.Write([]byte(dir))
}
