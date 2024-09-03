package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/master"
	"github.com/codecrafters-io/redis-starter-go/internal/slave"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
	"github.com/codecrafters-io/redis-starter-go/internal/transactions"
	"github.com/codecrafters-io/redis-starter-go/internal/utils"
)

func init() {
	f := &nested.Formatter{}
	log.SetFormatter(f)

	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
}

func main() {
	log.WithFields(log.Fields{
		"package":  "main",
		"function": "main",
	}).Info("An application has started!")

	port := flag.Int("port", 6379, "Port to listen on")
	replicaOf := flag.String("replicaof", "", "Replica to another server")
	dir := flag.String("dir", "", "Directory to store data")
	dbFileName := flag.String("dbfilename", "", "Database file name")

	flag.Parse()

	cfg := config.Config{
		Port:            *port,
		RedisDir:        *dir,
		RedisDbFileName: *dbFileName,
	}

	storeObj := store.NewStore()
	expiredCollector := store.NewExpiredCollector(storeObj)
	clients := clients.NewClients()
	transaction := transactions.NewTransaction()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "store", storeObj)
	ctx = context.WithValue(ctx, "clients", clients)
	ctx = context.WithValue(ctx, "transactions", transaction)

	address := fmt.Sprintf("0.0.0.0:%d", cfg.Port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Failed to bind to port ", cfg.Port)
		os.Exit(1)
	}
	defer l.Close()

	connChan := make(chan net.Conn)
	errChan := make(chan error)

	if *replicaOf == "" {
		cfg.Role = "master"
		cfg.Master = &config.Master{
			MasterReplId: "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
		}
	} else {
		cfg.Role = "slave"
		cfg.Slave = &config.Slave{
			Replicaof: *replicaOf,
		}
		masterConn, err := slave.ConnectMaster(*replicaOf, cfg)
		if err != nil {
			log.Fatalln("Error connecting to master: ", err)
		}

		reader, err := slave.Handshakes(masterConn, cfg)
		if err != nil {
			log.Fatalln("There is was error in handshakes with master : ", err)
		}

		go slave.ReadFromConnection(ctx, masterConn, reader, cfg)
	}

	utils.LoadRDB(ctx, cfg.RedisDir, cfg.RedisDbFileName)
	go master.AcceptConnections(l, connChan, errChan)
	go expiredCollector.Tick()

	for {
		select {
		case conn := <-connChan:

			transcationObj := transactions.GetTransactionsObj(ctx)
			transcationObj.AddConnection(conn)

			go master.ReadFromConnection(ctx, conn, cfg)

		case err := <-errChan:
			fmt.Println("Error accepting connection", err.Error())

		}
	}
}
