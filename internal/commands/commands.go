package commands

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
	"github.com/codecrafters-io/redis-starter-go/internal/transactions"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

type Command interface {
	Execute(ctx context.Context, conn net.Conn, config config.Config, args []string)
}

type CommandHandler func(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
)

var Propagated = [3]string{"SET", "DEL"}

var Commands = map[string]Command{
	"PING":     &PingCommand{},
	"ECHO":     &EchoCommand{},
	"SET":      &SetCommand{},
	"GET":      &GetCommand{},
	"INFO":     &InfoCommand{},
	"REPLCONF": &ReplConfCommand{},
	"PSYNC":    &PsyncCommand{},
	"WAIT":     &WaitCommand{},
	"CONFIG":   &ConfigCommand{},
	"KEYS":     &KeysCommand{},
	"INCR":     &IncrCommand{},
	"MULTI":    &MultiCommand{},
	"EXEC":     &ExecCommand{},
}

type ExecCommand struct{}

func (c *ExecCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
}

type MultiCommand struct{}

func (c *MultiCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	transactionObj := transactions.GetTransactionBufferObj(ctx)
	transactionObj.StartTransaction()

	conn.Write([]byte("+OK\r\n"))
}

type IncrCommand struct{}

func (c *IncrCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	if len(args) < 2 {
		log.Error("Missing arguments")
		return
	}
	key := args[1]

	storeObj := utils.GetStoreObj(ctx)

	value, err := storeObj.Incr(key)
	if err != nil {
		conn.Write([]byte("-ERR value is not an integer or out of range\r\n"))
		return
	}

	conn.Write([]byte(fmt.Sprintf(":%d\r\n", value)))
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
	switch config.Role {
	case "master":
		conn.Write([]byte("+PONG\r\n"))
	}
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

	switch config.Role {
	case "master":
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

		switch config.Role {
		case "master":
			master_replid := fmt.Sprintf("master_replid:%s", config.Master.MasterReplId)
			builder.WriteString(fmt.Sprintf("%s\n", master_replid))

			master_repl_offset := fmt.Sprintf(
				"master_repl_offset:%d",
				config.Master.MasterReplOffset.Load(),
			)
			builder.WriteString(
				fmt.Sprintf("%s\n", master_repl_offset),
			)
		}

		result := builder.String()

		finalResult := fmt.Sprintf("$%d\r\n%s\r\n", len(result), result)

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
	commands := map[string]CommandHandler{
		"master": c.handleMaster,
		"slave":  c.handleSlave,
	}

	if handler, exists := commands[config.Role]; exists {
		handler(ctx, conn, config, args)
	}
}

type PsyncCommand struct{}

func (c *PsyncCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	data := fmt.Sprintf(
		"+FULLRESYNC %s %d\r\n",
		config.Master.MasterReplId,
		config.Master.MasterReplOffset.Load(),
	)
	emptyRDB, _ := hex.DecodeString(redis.EMPTYRDBSTORE)
	data += fmt.Sprintf("$%d\r\n%s", len(emptyRDB), emptyRDB)

	_, err := conn.Write([]byte(data))
	if err != nil {
		fmt.Println("Error writing to ", conn.RemoteAddr().String())
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

type WaitCommand struct{}

func (c *WaitCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	if len(args) < 3 {
		fmt.Println("Not enough arguments")
		return
	}

	goal, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("Error converting goal:", err)
		return
	}

	timer, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println("Erro converting timer:", err)
		return
	}
	timerCh := time.After(time.Duration(timer) * time.Millisecond)

	clientsObj := utils.GetClientsObj(ctx)

	done := make(chan int, 1)

	var counter int64

	if config.Master.MasterReplOffset.Load() == 0 {
		done <- len(clientsObj.Clients)
	} else {

		cmdReplConf := redis.ConvertToRESP([]string{"REPLCONF", "GETACK", "*"})

		for _, client := range clientsObj.GetAll() {
			client.Write([]byte(cmdReplConf))
		}

		clientsObj.Subscribe(func(conn net.Conn, clientOffset int) {
			masterOffset := config.Master.MasterReplOffset.Load()
			log.WithFields(log.Fields{
				"package":      "commands",
				"function":     "WaitCommand.Execute",
				"masterOffset": masterOffset,
				"clientOffset": clientOffset,
				"conn":         conn,
			}).Info("Notification alert")

			if masterOffset <= int64(clientOffset) {
				atomic.AddInt64(&counter, 1)

				log.WithFields(log.Fields{
					"package":  "commands",
					"function": "WaitCommand.Execute",
					"value":    int(atomic.LoadInt64(&counter)),
					"goal":     goal,
				}).Info("Changing counter of acked clients")

				if goal == int(atomic.LoadInt64(&counter)) {
					done <- int(atomic.LoadInt64(&counter))
				}
			}
		})
	}

	writeMessage := func(c int) {
		message := fmt.Sprintf(":%d\r\n", c)
		if _, err := conn.Write([]byte(message)); err != nil {
			log.WithFields(log.Fields{
				"package":  "commands",
				"function": "WaitCommand.Execute",
				"error":    err,
			}).Error("Error writing to connection")
		}
	}

	for {
		select {
		case c := <-done:

			log.WithFields(log.Fields{
				"package":  "commands",
				"function": "WaitCommand.Execute",
				"value":    c,
			}).Info("Returning")
			writeMessage(c)

			return
		case <-timerCh:
			writeMessage(int(atomic.LoadInt64(&counter)))

			log.WithFields(log.Fields{
				"package":  "commands",
				"function": "WaitCommand.Execute",
				"value":    atomic.LoadInt64(&counter),
				"goal":     goal,
			}).Info("Time is up!")
			return
		}
	}
}

type ConfigCommand struct{}

func (c *ConfigCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	if len(args) < 3 {
		log.Error("Missing arguments")
		return
	}
	commands := map[string]CommandHandler{
		"GET": c.handleGet,
	}

	if handler, exists := commands[args[1]]; exists {
		handler(ctx, conn, config, args)
	}
}

type KeysCommand struct{}

func (c *KeysCommand) Execute(
	ctx context.Context,
	conn net.Conn,
	config config.Config,
	args []string,
) {
	if len(args) < 2 {
		log.Error("Missing arguments")
		return
	}

	commands := map[string]CommandHandler{
		"*": c.handleAll,
	}

	if handler, exists := commands[args[1]]; exists {
		handler(ctx, conn, config, args)
	}
}
