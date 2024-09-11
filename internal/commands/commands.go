package commands

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

var Propagated = [3]string{"SET", "DEL"}

type Command interface {
	Execute(ctx context.Context, conn net.Conn, config config.Config, args []string)
}

type EchoCommand struct{}

func (c *EchoCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	msg := args[1]
	conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), msg)))
}

type PingCommand struct{}

func (c *PingCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	conn.Write([]byte("+PONG\r\n"))
}

type SetCommand struct{}

func (c *SetCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	key, value := args[1], args[2]

	var px *int

	if len(args) > 3 {
		switch strings.ToUpper(args[3]) {
		case "PX":
			parsedPx, err := strconv.Atoi(args[4])
			if err != nil {
				conn.Write([]byte("px arg in not valid"))
				return
			}
			px = &parsedPx
		}
	}

	storeFromContext := ctx.Value("store")

	if storeFromContext != nil {
		if store, ok := storeFromContext.(*store.Store); !ok {
			log.Fatalf("Expected *store.Store, got %T", storeFromContext)
		} else {
			store.Set(key, value, px)
		}
	}

	if config.Role == "master" {
		conn.Write([]byte("+OK\r\n"))
	}
}

type GetCommand struct{}

func (c *GetCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	key := args[1]

	storeFromContext := ctx.Value("store")

	if storeFromContext != nil {
		if store, ok := storeFromContext.(*store.Store); !ok {
			log.Fatalf("Expected *store.Store, got %T", storeFromContext)
		} else {
			value, err := store.Get(key)
			if err != nil {
				conn.Write([]byte("$-1\r\n"))
			} else {
				conn.Write([]byte(fmt.Sprintf("+%s\r\n", value)))
			}
		}
	}
}

type InfoCommand struct{}

func (c *InfoCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	switch args[1] {
	case "replication":
		var builder strings.Builder
		builder.Grow(128)

		role := fmt.Sprintf("role:%s", config.Role)
		builder.WriteString(fmt.Sprintf("%s\n", role))

		master_replid := fmt.Sprintf("master_replid:%s", config.MasterReplId)
		builder.WriteString(fmt.Sprintf("%s\n", master_replid))

		master_repl_offset := fmt.Sprintf("master_repl_offset:%d", config.MasterReplOffset)
		builder.WriteString(
			fmt.Sprintf("%s\n", master_repl_offset),
		)

		result := builder.String()

		finalResult := fmt.Sprintf("$%d\r\n%s\r\n", len(result), result)
		fmt.Println(finalResult)

		conn.Write([]byte(finalResult))

	default:
		conn.Write([]byte("-Error\r\n"))
	}
}

type ReplConfCommand struct{}

func (c *ReplConfCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	switch config.Role {
	case "master":
		switch args[1] {
		case "ACK":
			fmt.Println("REPL ACK ", args[2])
		case "capa":
			conn.Write([]byte("+OK\r\n"))
		case "listening-port":
			conn.Write([]byte("+OK\r\n"))
		}
	case "slave":
		if args[1] == "GETACK" && args[2] == "*" {
			conn.Write([]byte("*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$1\r\n0\r\n"))
		}
	}
}

type PsyncCommand struct{}

func (c *PsyncCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	data := fmt.Sprintf("+FULLRESYNC %s %d\r\n", config.MasterReplId, config.MasterReplOffset)
	emptyRDB, _ := hex.DecodeString(redis.EMPTYRDBSTORE)
	data += fmt.Sprintf("$%d\r\n%s", len(emptyRDB), emptyRDB)

	n, err := conn.Write([]byte(data))
	if err != nil {
		fmt.Println("Error writing to ", conn.RemoteAddr().String())
	} else {
		fmt.Printf("Wrote %d bytes to the buffer\r\n", n)
	}

	clientsFromContext := ctx.Value("clients")
	if clientsFromContext != nil {
		if clients, ok := clientsFromContext.(*clients.Clients); !ok {
			log.Fatalf("Expected *master.Clients, got %T", clientsFromContext)
		} else {
			clients.Set(conn)
		}
	}
}

var Commands = map[string]Command{
	"PING":     &PingCommand{},
	"ECHO":     &EchoCommand{},
	"SET":      &SetCommand{},
	"GET":      &GetCommand{},
	"INFO":     &InfoCommand{},
	"REPLCONF": &ReplConfCommand{},
	"PSYNC":    &PsyncCommand{},
}
