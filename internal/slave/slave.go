package slave

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
)

type CommandRequest struct {
	args   []string
	offset int
}

func HandleSlaveMode(ctx context.Context, config interfaces.IConfig) {
	masterConn, err := ConnectMaster(config)
	if err != nil {
		log.Fatalln("Error connecting to master: ", err)
	}

	reader, err := Handshakes(masterConn, config)
	if err != nil {
		log.Fatalln("There is was error in handshakes with master : ", err)
	}

	go ReadFromConnection(ctx, masterConn, reader, config)
}

func ReadFromConnection(
	ctx context.Context,
	conn net.Conn,
	reader *bufio.Reader,
	config interfaces.IConfig,
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
	config interfaces.IConfig,
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
		config.GetSlave().AddOffset(int64(cmdRequest.offset))

		fmt.Printf("Total offset after command %d\r\n", config.GetSlave().GetOffset())
	}
}
