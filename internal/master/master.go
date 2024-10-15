package master

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/interfaces"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

func CreateListener(config interfaces.IConfig) net.Listener {
	address := fmt.Sprintf("0.0.0.0:%d", config.GetPort())

	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Failed to bind to port ", config.GetPort())
		os.Exit(1)
	}
	return l
}

func AcceptConnections(l net.Listener, connChan chan<- net.Conn, errChan chan<- error) {
	for {
		conn, err := l.Accept()
		if err != nil {

			errChan <- err
			return
		}

		connChan <- conn
	}
}

func HandleConnections(
	ctx context.Context,
	connChan chan net.Conn,
	errChan chan error,
	config interfaces.IConfig,
) {
	for {
		select {
		case conn := <-connChan:
			HandleNewConnection(ctx, conn, config)

		case err := <-errChan:
			fmt.Println("Error accepting connection", err.Error())

		}
	}
}

func HandleNewConnection(ctx context.Context, conn net.Conn, config interfaces.IConfig) {
	transactionObj, ok := utils.GetFromCtx[*commands.Transactions](ctx, "transactions")

	if !ok {
		log.Error("No transactions in context")
		return
	}
	transactionObj.AddConnection(conn)

	go ReadFromConnection(ctx, conn, config)
}

func ReadFromConnection(ctx context.Context, conn net.Conn, config interfaces.IConfig) {
	defer conn.Close()

	for {
		r := bufio.NewReader(conn)

		args, _, err := redis.UnpackInput(r)
		if err != nil {
			break
		}

		if len(args) == 0 {
			break
		}

		go HandleCommand(ctx, conn, config, args)
	}
}

func HandleCommand(ctx context.Context, conn net.Conn, config interfaces.IConfig, args []string) {
	cmd, exists := commands.Commands[strings.ToUpper(args[0])]
	if !exists {
		conn.Write([]byte("-Error\r\n"))
		return
	}

	baseCommandHandler := &BaseCommandHandler{}
	discardConditionHandler := &DiscardConditionHandler{}
	queuedConditionHandler := &QueuedConditionHandler{}

	baseCommandHandler.SetNext(discardConditionHandler)
	discardConditionHandler.SetNext(queuedConditionHandler)

	if !baseCommandHandler.Handle(ctx, conn, config, args, cmd) {
		return
	}

	cmd.Execute(ctx, conn, config, args)

	for _, command := range commands.Propagated {
		if command == args[0] {
			cmd := redis.ConvertToRESP(
				args,
			)
			logrus.Info("Master: ", config.GetMaster())
			config.GetMaster().AddMasterReplOffset(int64(len(cmd)))

			SendCommandAllClients(ctx, conn, config, cmd)
		}
	}
}

func SendCommandAllClients(
	ctx context.Context,
	conn net.Conn,
	config interfaces.IConfig,
	cmd string,
) {
	clients, ok := utils.GetFromCtx[*clients.Clients](ctx, "clients")

	if !ok {
		log.Error("No store in context")
		return
	}

	for _, clientConn := range clients.GetAll() {
		log.WithFields(log.Fields{
			"package":  "master",
			"function": "HandleCommand",
		}).Info()

		clientConn.Write([]byte(cmd))
	}
}
