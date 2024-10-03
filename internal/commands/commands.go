package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/redis"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
	"github.com/codecrafters-io/redis-starter-go/internal/transactions"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

type Command interface {
	Execute(ctx context.Context, conn io.Writer, config config.Config, args []string)
}

type CommandHandler func(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
)

var Propagated = [3]string{"SET", "DEL"}

var Commands = map[string]Command{
	"PING": &PingCommand{},
	"ECHO": &EchoCommand{},
	"SET":  &SetCommand{},
	"GET":  &GetCommand{},

	"INFO":     &InfoCommand{},
	"REPLCONF": &ReplConfCommand{},
	"PSYNC":    &PsyncCommand{},
	"WAIT":     &WaitCommand{},

	"CONFIG": &ConfigCommand{},
	"KEYS":   &KeysCommand{},
	"INCR":   &IncrCommand{},

	"MULTI":   &MultiCommand{},
	"EXEC":    &ExecCommand{},
	"DISCARD": &DiscardCommand{},

	"TYPE":   &TypeCommand{},
	"XADD":   &XAddCommand{},
	"XREAD":  &XReadCommand{},
	"XRANGE": &XRangeCommand{},
}

/*
The XADD command adds a new entry to a stream.
*/
type XAddCommand struct{}

func (c *XAddCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	if len(args) < 3 {
		log.Error("Missing arguments")
		return
	}

	var answerStr string

	storeObj := utils.GetStoreObj(ctx)

	key := args[1]

	id, err := store.FormID(key, args[2], storeObj)

	fields := make(map[string]string)

	for i := 3; i < len(args); i += 2 {
		fields[args[i]] = args[i+1]
	}

	if err != nil {
		answerStr = fmt.Sprintf("-ERR %s\r\n", err.Error())
	} else {
		answerStr = fmt.Sprintf("$%d\r\n%s\r\n", len(id), id)

		streamMessage := store.StreamMessage{
			ID:     id,
			Fields: fields,
		}

		storeObj.XAdd(key, streamMessage)
	}

	logrus.Debug("XADD BEFORE SELECT")

	blockCh := utils.GetBlockChObj(ctx)

	select {
	case blockCh <- struct{}{}:
	default:
	}

	logrus.Debug("XADD AFTER SELECT")

	conn.Write([]byte(answerStr))
}

type XReadCommand struct{}

func (c *XReadCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	var streamsIndex int
	var numStreams int
	var block bool

	if args[1] == "block" {

		streamsIndex = 4

		block = true

		numStreams = (len(args) - 4) / 2

		timeSleep, err := strconv.Atoi(args[2])
		if err != nil {
			logrus.Error(err)
			return
		}

		time.Sleep(time.Duration(timeSleep) * time.Millisecond)

		if args[2] == "0" {
			blockCh := utils.GetBlockChObj(ctx)
			<-blockCh

			// блокируем горутину, пока не получим сообщения XADD из другого потока
			// получаем из ctx канал и ждем из него сообщения
			// в XADD добавляем значение в канал
			// тем самым разблокируем эту горутину
			// но как быть если XADD нужно не всегда что-то отправлять в канал ?
			// select {} где case ch <- "XADD" либо default
		}

	} else {
		streamsIndex = 2

		numStreams = (len(args) - 2) / 2
	}

	streamKeys := args[streamsIndex : streamsIndex+numStreams]
	IDs := args[streamsIndex+numStreams:]

	storeObj := utils.GetStoreObj(ctx)

	var bb bytes.Buffer

	type StreamPair struct {
		streamKey string
		id        string
		messages  []store.StreamMessage
	}

	streamPairs := make([]StreamPair, 0, len(streamKeys))

	for i := range streamKeys {
		streamPairs = append(streamPairs, StreamPair{
			streamKey: streamKeys[i],
			id:        IDs[i],
			messages:  make([]store.StreamMessage, 0, 8),
		})
	}

	for index, streamPair := range streamPairs {

		messages, err := storeObj.GetStreamsExclusive(streamPair.streamKey, streamPair.id)
		if err != nil {
			logrus.Error(err)
			return
		}

		streamPairs[index].messages = messages

		if block && len(messages) == 0 {
			conn.Write([]byte("$-1\r\n"))
			return
		}
	}

	bb.WriteString(arrayResp(len(streamKeys)))

	for _, streamPair := range streamPairs {
		writeStreamMessage(&bb, streamPair.streamKey, streamPair.messages)
	}

	logrus.Error(bb.String())

	conn.Write(bb.Bytes())
}

/*
The XRANGE command returns a range of elements from a stream.
*/
type XRangeCommand struct{}

func (c *XRangeCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	if len(args) < 3 {
		log.Error("Missing arguments")
		return
	}

	key := args[1]
	IDs := args[2:4]

	storeObj := utils.GetStoreObj(ctx)
	res, err := storeObj.GetStreamsRange(key, [2]string{IDs[0], IDs[1]})
	if err != nil {
		logrus.Error(err)
		return
	}

	var bb bytes.Buffer

	bb.Write([]byte(fmt.Sprintf("*%d\r\n", len(res))))

	for _, v := range res {
		bb.Write([]byte(fmt.Sprintf("*2\r\n")))
		bb.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v.ID), v.ID)))
		bb.Write([]byte(fmt.Sprintf("*%d\r\n", len(v.Fields)*2)))

		for k, v := range v.Fields {
			logrus.Error(k + ": " + v)
			bb.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(k), k)))
			bb.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)))
		}
	}

	conn.Write(bb.Bytes())
}

/*
The TYPE command returns the type of value stored at a given key.
*/
type TypeCommand struct{}

func (c *TypeCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	if len(args) < 2 {
		log.Error("Missing arguments")
		return
	}

	key := args[1]

	storeObj := utils.GetStoreObj(ctx)

	keyType, err := storeObj.GetType(key)
	if err != nil {
		conn.Write([]byte("+none\r\n"))
		return
	}

	conn.Write([]byte(fmt.Sprintf("+%s\r\n", keyType)))
}

/*
The DISCARD command discards all commands issued after MULTI.
*/
type DiscardCommand struct{}

func (c *DiscardCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	transactionsObj := transactions.GetTransactionsObj(ctx)

	if conn, ok := conn.(net.Conn); ok {
		transactionBufferObj := transactionsObj.GetTransactionBuffer(conn)

		if !transactionBufferObj.IsTransactionActive() {
			conn.Write([]byte("-ERR DISCARD without MULTI\r\n"))
			return
		}

		transactionBufferObj.DiscardTransaction()
		conn.Write([]byte("+OK\r\n"))
	}
}

/*
The EXEC command executes all the previously queued commands issued with MULTI.
*/
type ExecCommand struct{}

func (c *ExecCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	transactionsObj := transactions.GetTransactionsObj(ctx)

	var buffer bytes.Buffer
	buffer.Grow(8)

	var lenCommands int

	if conn, ok := conn.(net.Conn); ok {
		transactionBufferObj := transactionsObj.Values[conn]

		if !transactionBufferObj.IsTransactionActive() {
			conn.Write([]byte("-ERR EXEC without MULTI\r\n"))
			return
		}

		if transactionBufferObj.IsBufferEmpty() {
			conn.Write([]byte("*0\r\n"))

			transactionBufferObj.InActivateTransaction()
			return
		}

		commands := transactionBufferObj.PopCommands()
		lenCommands = len(commands)

		for _, command := range commands {
			args := command.Args
			cmd := command.CMD

			if _, ok := conn.(net.Conn); ok {
				cmd.Execute(ctx, &buffer, config, args)
			}
		}

		transactionBufferObj.InActivateTransaction()
	}

	result := fmt.Sprintf("*%d\r\n%s", lenCommands, buffer.String())
	conn.Write([]byte(result))
	return
}

/*
The MULTI command marks the start of a transaction block.
*/
type MultiCommand struct{}

func (c *MultiCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	transactionsObj := transactions.GetTransactionsObj(ctx)

	if conn, ok := conn.(net.Conn); ok {
		transactionBufferObj := transactionsObj.Values[conn]
		transactionBufferObj.StartTransaction()
	}

	conn.Write([]byte("+OK\r\n"))
}

/*
The INCR command increments the number stored at key by one.
*/
type IncrCommand struct{}

func (c *IncrCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The ECHO command returns a line of text to the client.
*/
type EchoCommand struct{}

func (c *EchoCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	msg := args[1]
	conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), msg)))
}

/*
The PING command returns PONG.
*/
type PingCommand struct{}

func (c *PingCommand) Execute(
	ctx context.Context,
	conn io.Writer,
	config config.Config,
	args []string,
) {
	switch config.Role {
	case "master":
		conn.Write([]byte("+PONG\r\n"))
	}
}

/*
The SET command sets the string value of a key.
*/
type SetCommand struct{}

func (c *SetCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The GET command returns the value associated with a key.
*/
type GetCommand struct{}

func (c *GetCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The INFO command returns information and statistics about the server.
*/
type InfoCommand struct{}

func (c *InfoCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The REPLCONF command sets the configuration of the replication link.
*/
type ReplConfCommand struct{}

func (c *ReplConfCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The PSYNC command is used to synchronize replication.
*/
type PsyncCommand struct{}

func (c *PsyncCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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
		fmt.Println("Error writing", err)
	}

	clientsFromContext := ctx.Value("clients")
	if clientsFromContext != nil {
		if clients, ok := clientsFromContext.(*clients.Clients); !ok {
			log.Fatalf("Expected *master.Clients, got %T", clientsFromContext)
		} else {
			if conn, ok := conn.(net.Conn); ok {
				clients.Set(conn)
			}
		}
	}
}

/*
The WAIT command is used to wait for replication.
*/
type WaitCommand struct{}

func (c *WaitCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The CONFIG command is used to get or set configuration parameters.
*/
type ConfigCommand struct{}

func (c *ConfigCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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

/*
The KEYS command returns all keys that match the given pattern.
*/
type KeysCommand struct{}

func (c *KeysCommand) Execute(
	ctx context.Context,
	conn io.Writer,
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
