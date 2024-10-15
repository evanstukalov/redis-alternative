package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
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
