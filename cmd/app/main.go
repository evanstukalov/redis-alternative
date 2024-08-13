package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/clients"
	"github.com/codecrafters-io/redis-starter-go/internal/config"
	"github.com/codecrafters-io/redis-starter-go/internal/master"
	"github.com/codecrafters-io/redis-starter-go/internal/slave"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func main() {
	port := flag.Int("port", 6379, "Port to listen on")
	replicaOf := flag.String("replicaof", "", "Replica to another server")

	flag.Parse()

	cfg := config.Config{
		Port: *port,
	}

	storeObj := store.NewStore()
	expiredCollector := store.NewExpiredCollector(storeObj)
	defer expiredCollector.Stop()
	clients := clients.NewClients()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "store", storeObj)
	ctx = context.WithValue(ctx, "clients", clients)

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
			MasterReplId:     "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
			MasterReplOffset: 0,
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

	go master.AcceptConnections(l, connChan, errChan)
	go expiredCollector.Tick()

	for {
		select {
		case conn := <-connChan:

			go master.ReadFromConnection(ctx, conn, cfg)

		case err := <-errChan:
			fmt.Println("Error accepting connection", err.Error())

		}
	}
}
