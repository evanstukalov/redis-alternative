package master

import (
	"bufio"
	"context"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/commands"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/transactions"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

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

func ReadFromConnection(ctx context.Context, conn net.Conn, config config.Config) {
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

func HandleCommand(ctx context.Context, conn net.Conn, config config.Config, args []string) {
	cmd, exists := commands.Commands[strings.ToUpper(args[0])]
	if !exists {
		conn.Write([]byte("-Error\r\n"))
		return
	}

	log.WithFields(log.Fields{
		"package":  "master",
		"function": "HandleCommand",
		"cmd":      args,
		"conn":     conn.RemoteAddr().String(),
	}).Info()

	transactionsObj := transactions.GetTransactionBufferObj(ctx)

	if _, ok := cmd.(*commands.ExecCommand); ok {
		if !transactionsObj.IsTransactionActive() {
			conn.Write([]byte("-ERR EXEC without MULTI\r\n"))
			return
		}

		if transactionsObj.IsBufferEmpty() {
			conn.Write([]byte("*0\r\n"))

			transactionsObj.EndTransaction()
			return
		}

		commands := transactionsObj.PopCommands()

		for _, command := range commands {
			args := command.Args
			cmd := command.CMD
			cmd.Execute(ctx, conn, config, args)
		}

		transactionsObj.EndTransaction()

		conn.Write([]byte("+OK\r\n"))
		return

	}

	if transactionsObj.IsTransactionActive() {
		transactionsObj.PutCommand(&transactions.BufferedCommand{
			CMD:  cmd,
			Args: args,
		})

		conn.Write([]byte("+QUEUED\r\n"))
		return
	}

	cmd.Execute(ctx, conn, config, args)

	for _, command := range commands.Propagated {
		if command == args[0] {
			cmd := redis.ConvertToRESP(
				args,
			)
			config.Master.MasterReplOffset.Add(int64(len(cmd)))

			SendCommandAllClients(ctx, conn, config, cmd)
		}
	}
}

func SendCommandAllClients(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	cmd string,
) {
	clients := utils.GetClientsObj(ctx)

	for _, clientConn := range clients.GetAll() {
		log.WithFields(log.Fields{
			"package":  "master",
			"function": "HandleCommand",
		}).Info()

		clientConn.Write([]byte(cmd))
	}
}
