package slave

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
)

type CommandRequest struct {
	args   []string
	offset int
}

func ReadFromConnection(
	ctx context.Context,
	conn net.Conn,
	reader *bufio.Reader,
	config config.Config,
) {
	defer conn.Close()

	commandChannel := make(chan CommandRequest, 64)
	go HandleCommand(ctx, conn, config, commandChannel)

	for {
		args, offset, err := redis.UnpackInput(reader)
		if err != nil {
			break
		}

		fmt.Println("New command from master :", args)

		if len(args) == 0 {
			break
		}

		commandChannel <- CommandRequest{args: args, offset: offset}
	}
	close(commandChannel)
}

func HandleCommand(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	commandChannel <-chan CommandRequest,
) {
	for cmdRequest := range commandChannel {

		cmd, exists := commands.Commands[strings.ToUpper(cmdRequest.args[0])]
		if !exists {
			conn.Write([]byte("-Error\r\n"))
			return
		}
		fmt.Printf("Offset new command: %d\r\n", cmdRequest.offset)
		cmd.Execute(ctx, conn, config, cmdRequest.args)
		config.Slave.Offset.Add(int64(cmdRequest.offset))

		fmt.Printf("Total offset after command %d\r\n", config.Slave.Offset.Load())
	}
}
