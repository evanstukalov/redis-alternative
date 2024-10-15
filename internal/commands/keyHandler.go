package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
)

func (c *KeysCommand) handleAll(
	ctx context.Context,
	conn io.Writer,
	config interfaces.IConfig,
	args []string,
) {
	fileContent := redis.ReadFile(config.GetRedisDir() + "/" + config.GetRedisDbFileName())
	conn.Write([]byte(fmt.Sprintf("*1\r\n$%d\r\n%s\r\n", len(fileContent), fileContent)))
}
