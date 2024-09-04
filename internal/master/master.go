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
	log.WithFields(log.Fields{
		"package":  "master",
		"function": "HandleCommand",
		"cmd":      args,
		"conn":     conn.RemoteAddr().String(),
	}).Info()

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
